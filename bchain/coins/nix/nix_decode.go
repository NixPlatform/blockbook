package nix

import (
   "bytes"
   "encoding/binary"
   "io"
   "time"
   "math"

   "github.com/btcsuite/btcd/chaincfg/chainhash"
   "github.com/btcsuite/btcd/wire"

   "fmt"
)

const (
   minTxInPayload = 41
   MinTxOutPayload = 9
   maxTxInPerMessage = (wire.MaxMessagePayload / minTxInPayload) + 1
   maxTxOutPerMessage = (wire.MaxMessagePayload / MinTxOutPayload) + 1
   maxWitnessItemsPerInput = 500000

   // output Types
   OUTPUT_NULL             = 0 // marker for CCoinsView (0.14)
   OUTPUT_STANDARD         = 1
)

// Extend Wire structs
type Tx struct {
   msg *wire.MsgTx
   outTypes []uint8
}

type TxBlock struct  {
   txs []*Tx
}

// -------- UNSERIALIZATION (replaces BtcDecode/Deserialize) --------
// Decode Block
func UnserializeBlock(msg *wire.MsgBlock, blk *TxBlock, hashWitnessMerkleRoot *chainhash.Hash,
      hashAccumulators *chainhash.Hash, r io.Reader) error {

   err := readBlockHeader(r, &msg.Header, hashWitnessMerkleRoot, hashAccumulators)

	if err != nil {
		return err
	}

	txCount, err := ReadVarInt(r)

	if err != nil {
		return err
	}

	msg.Transactions = make([]*wire.MsgTx, 0, txCount)
   blk.txs = make([]*Tx, 0, txCount)

	for i := uint64(0); i < txCount; i++ {
		txmess := wire.MsgTx{}
      tx := Tx{}
		err := UnserializeTx(&txmess, &tx, r)
		if err != nil {
			return err
		}
		msg.Transactions = append(msg.Transactions, &txmess)
      blk.txs = append(blk.txs, &tx)
	}

	return nil
}

// Decode Tx
func UnserializeTx(msg *wire.MsgTx, tx *Tx, r io.Reader) error {
   *msg = wire.MsgTx{}
   *tx = Tx{}

   tx.msg = msg

   // Tx Version + Tx Type
   version, err := Uint16(r, binary.LittleEndian)
   if err != nil {
      return err
   }
   msg.Version = int32(version)

   // Read the fUseSegwit flag value
   fUseSegwit := false
   err = readElement(r, &fUseSegwit)
	if err != nil {
		return err
	}

   // Read nLockTime
   nLockTime, err := Uint32(r, binary.LittleEndian)
   if err != nil {
      return err
   }
   msg.LockTime = nLockTime

   // Read the txin count
   count, err := ReadVarInt(r)
	if err != nil {
		return err
	}

   // A count of zero (meaning no TxIn's to the uninitiated) indicates
	// this is a transaction with witness data.
   var flag [1]byte
   if count == 0 && fUseSegwit {
      // We need to read the flag, which is a single byte.
   	if _, err = io.ReadFull(r, flag[:]); err != nil {
   		return err
   	}
      // At the moment, the flag MUST be 0x01. In the future other
   	// flag types may be supported.
   	if flag[0] != 0x01 {
   		str := fmt.Sprintf("witness tx but flag byte is %x", flag)
   		return messageError("UnserializeTx", str)
   	}
   	// With the Segregated Witness specific fields decoded, we can
   	// now read in the actual txin count.
   	count, err = ReadVarInt(r)
   	if err != nil {
   		return err
   	}
   }

   // Prevent more input transactions than could possibly fit into a
	// message.  It would be possible to cause memory exhaustion and panics
	// without a sane upper bound on this count.
	if count > uint64(maxTxInPerMessage) {
		str := fmt.Sprintf("too many input transactions to fit into "+
			"max message size [count %d, max %d]", count,
			maxTxInPerMessage)
		return messageError("UnserializeTx", str)
	}

   // Deserialize the inputs.
   var totalScriptSize uint64
   txIns := make([]wire.TxIn, count)
   msg.TxIn = make([]*wire.TxIn, count)
   for i := uint64(0); i < count; i++ {
		ti := &txIns[i]
		msg.TxIn[i] = ti
		err = readTxIn(r, ti)
		if err != nil {
			return err
		}
		totalScriptSize += uint64(len(ti.SignatureScript))
	}

   // Read the txout count
   count, err = ReadVarInt(r)
	if err != nil {
		return err
	}

   // Prevent more output transactions than could possibly fit into a
	// message.  It would be possible to cause memory exhaustion and panics
	// without a sane upper bound on this count.
	if count > uint64(maxTxOutPerMessage) {
		str := fmt.Sprintf("too many output transactions to fit into "+
			"max message size [count %d, max %d]", count,
			maxTxOutPerMessage)
		return messageError("UnserializeTx", str)
	}

   // Deserialize the outputs.
	txOuts := make([]wire.TxOut, count)
   tx.outTypes = make([]uint8, count)
	msg.TxOut = make([]*wire.TxOut, count)
	for i := uint64(0); i < count; i++ {
		to := &txOuts[i]
		msg.TxOut[i] = to

		outType, err := readTxOut(r, to)
		if err != nil {
			return err
		}
      tx.outTypes[i] = outType
		totalScriptSize += uint64(len(to.PkScript))
	}

   // Read witness data if present
   if fUseSegwit  {
      for _, txin := range msg.TxIn {
         // For each input, the witness is encoded as a stack
         // with one or more items. Therefore, we first read a
         // varint which encodes the number of stack items.
         witCount, err := ReadVarInt(r)
         if err != nil {
				return err
			}

         // Prevent a possible memory exhaustion attack by
			// limiting the witCount value to a sane upper bound.
			if witCount > maxWitnessItemsPerInput {
				str := fmt.Sprintf("too many witness items to fit "+
					"into max message size [count %d, max %d]",
					witCount, maxWitnessItemsPerInput)
				return messageError("MsgTx.BtcDecode", str)
			}

         // Then for witCount number of stack items, each item
			// has a varint length prefix, followed by the witness
			// item itself.
			txin.Witness = make([][]byte, witCount)
			for j := uint64(0); j < witCount; j++ {
				txin.Witness[j], err = readScript(r)
				if err != nil {
					return err
				}
				totalScriptSize += uint64(len(txin.Witness[j]))
			}
      }
   }

   return nil
}


// -------- SERIALIZATION (replaces BtcEncode/Serialize) --------

// TxHash generates the Hash for the transaction.
func TxHash(msg *wire.MsgTx, tx *Tx) chainhash.Hash {
	// Encode the transaction and calculate double sha256 on the result.
	// Ignore the error returns since the only way the encode could fail
	// is being out of memory or due to nil pointers, both of which would
	// cause a run-time panic.
	buf := bytes.NewBuffer(make([]byte, 0, baseSize(msg)))
	_ = SerializeTxNoWitness(msg, tx, buf)
	return chainhash.DoubleHashH(buf.Bytes())
}

// SerializeNoWitness encodes the transaction to w in an identical manner to
// Serialize, however even if the source transaction has inputs with witness
// data, the old serialization format will still be used.
func SerializeTxNoWitness(msg *wire.MsgTx, tx *Tx, w io.Writer) error {
	return SerializeTx(msg, w, false, tx)
}

// EncodeTx
func SerializeTx(msg *wire.MsgTx, w io.Writer, witnessEncoding bool, tx *Tx) error {
   // Tx Version + Tx Type
	err := PutUint16(w, binary.LittleEndian, uint16(msg.Version))
	if err != nil {
		return err
	}

   // Write the fUseSegwit flag value
   err = writeElement(w, &witnessEncoding)
	if err != nil {
		return err
	}

   // Write nLockTime
   err = PutUint32(w, binary.LittleEndian, uint32(msg.LockTime))
   if err != nil {
      return err
   }

   // Write txin count
   count := uint64(len(msg.TxIn))
	err = WriteVarInt(w, count)
	if err != nil {
		return err
	}

   // !TODO: segwit

   // Write inputs
   for _, ti := range msg.TxIn {
		err = writeTxIn(w, ti)
		if err != nil {
			return err
		}
	}

   // Write txout count
   count = uint64(len(msg.TxOut))
	err = WriteVarInt(w, count)
	if err != nil {
		return err
	}

   // Write outputs
   for i, to := range msg.TxOut {
		err = writeTxOut(w, to, tx.outTypes[i])
		if err != nil {
			return err
		}
	}

   // !TODO: witness data
   return nil
}

// --- READ METHODS

// readBlockHeader reads a bitcoin block header from r.  See Deserialize for
// decoding block headers stored to disk, such as in a database, as opposed to
// decoding from the wire.
func readBlockHeader(r io.Reader, bh *wire.BlockHeader,
      hwmr *chainhash.Hash, hacc *chainhash.Hash) error {
	return readElements(r, &bh.Version, &bh.PrevBlock, &bh.MerkleRoot, hwmr,
		(*time.Time)(&bh.Timestamp), &bh.Bits, &bh.Nonce, hacc)
}

// readScript reads a variable length byte array that represents a transaction
// script.  It is encoded as a varInt containing the length of the array
// followed by the bytes themselves.
func readScript(r io.Reader) ([]byte, error) {
	count, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	b := make([]byte, count)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// readOutPoint reads the next sequence of bytes from r as an OutPoint.
func readOutPoint(r io.Reader, op *wire.OutPoint) error {
	_, err := io.ReadFull(r, op.Hash[:])
	if err != nil {
		return err
	}

	op.Index, err = Uint32(r, binary.LittleEndian)
	return err
}

// readTxIn reads the next sequence of bytes from r as a transaction input
// (TxIn).
func readTxIn(r io.Reader, ti *wire.TxIn) error {
	err := readOutPoint(r, &ti.PreviousOutPoint)
	if err != nil {
		return err
	}

	ti.SignatureScript, err = readScript(r)
	if err != nil {
		return err
	}

	return readElement(r, &ti.Sequence)
}

// readTxOut reads the next sequence of bytes from r as a transaction output
// (TxOut).
func readTxOut(r io.Reader, to *wire.TxOut) (uint8, error) {
   // initialize value at zero
   to.Value = 0
   // read TxOut type
   outType, err := Uint8(r)
   if err != nil {
      return 0, err
   }

   switch outType {
   default: // OUTPUT_STANDARD
      err = readElement(r, &to.Value)
   	if err != nil {
   		break
   	}
      to.PkScript, err = readScript(r)
   }
   return outType, err
}

// Uint8 reads one byte from the provided reader and
// returns the resulting uint8.
func Uint8(r io.Reader) (uint8, error) {
   buf := make([]byte, 1)

   if _, err := io.ReadFull(r, buf); err != nil {
      return 0, err
   }

   rv := buf[0]
   return rv, nil
}

// Uint16 reads two bytes from the provided reader, converts it
// to a number using the provided byte order, and returns the resulting uint16.
func Uint16(r io.Reader, byteOrder binary.ByteOrder) (uint16, error) {
   buf := make([]byte, 2)

   if _, err := io.ReadFull(r, buf); err != nil {
      return 0, err
   }

   rv := byteOrder.Uint16(buf)
   return rv, nil
}

// Uint32 reads four bytes from the provided reader, converts it
// to a number using the provided byte order, and returns the resulting uint32.
func Uint32(r io.Reader, byteOrder binary.ByteOrder) (uint32, error) {
   buf := make([]byte, 4)

   if _, err := io.ReadFull(r, buf); err != nil {
      return 0, err
   }

   rv := byteOrder.Uint32(buf)
   return rv, nil
}

// Uint64 reads eight bytes from the provided reader, converts it
// to a number using the provided byte order, and returns the resulting uint64.
func Uint64(r io.Reader, byteOrder binary.ByteOrder) (uint64, error) {
   buf := make([]byte, 8)

   if _, err := io.ReadFull(r, buf); err != nil {
      return 0, err
   }

   rv := byteOrder.Uint64(buf)
   return rv, nil
}

// ReadVarInt reads a variable length integer from r and returns it as a uint64.
func ReadVarInt(r io.Reader) (uint64, error) {
   discriminant, err := Uint8(r)
	if err != nil {
		return 0, err
	}

   var rv uint64
	switch discriminant {
	case 0xff:
		sv, err := Uint64(r, binary.LittleEndian)
		if err != nil {
			return 0, err
		}
		rv = sv
		// The encoding is not canonical if the value could have been
		// encoded using fewer bytes.
		min := uint64(0x100000000)
		if rv < min {
			return 0, messageError("ReadVarInt", "rv < min")
		}

	case 0xfe:
		sv, err := Uint32(r, binary.LittleEndian)
		if err != nil {
			return 0, err
		}
		rv = uint64(sv)
		// The encoding is not canonical if the value could have been
		// encoded using fewer bytes.
		min := uint64(0x10000)
		if rv < min {
         return 0, messageError("ReadVarInt", "rv < min")
		}

	case 0xfd:
		sv, err := Uint16(r, binary.LittleEndian)
		if err != nil {
			return 0, err
		}
		rv = uint64(sv)
		// The encoding is not canonical if the value could have been
		// encoded using fewer bytes.
		min := uint64(0xfd)
		if rv < min {
			return 0, messageError("ReadVarInt", "rv < min")
		}

	default:
		rv = uint64(discriminant)
	}

	return rv, nil
}

// readElement reads the next sequence of bytes from r using little endian
// depending on the concrete type of element pointed to.
func readElement(r io.Reader, element interface{}) error {
	// Attempt to read the element based on the concrete type via fast
	// type assertions first.
	switch e := element.(type) {
	case *int32:
		rv, err := Uint32(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = int32(rv)
		return nil

	case *uint32:
		rv, err := Uint32(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil

	case *int64:
		rv, err := Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = int64(rv)
		return nil

	case *uint64:
		rv, err := Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil

	case *bool:
		rv, err := Uint8(r)
		if err != nil {
			return err
		}
		if rv == 0x00 {
			*e = false
		} else {
			*e = true
		}
		return nil

   // Unix timestamp encoded as a uint32.
   case *time.Time:
		rv, err := Uint32(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = time.Unix(int64(rv), 0)
		return nil

   // Hash.
   case *chainhash.Hash:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil
   }

	// Fall back to the slower binary.Read if a fast path was not available
	// above.
	return binary.Read(r, binary.LittleEndian, element)
}

// readElements reads multiple items from r.  It is equivalent to multiple
// calls to readElement.
func readElements(r io.Reader, elements ...interface{}) error {
	for _, element := range elements {
		err := readElement(r, element)
		if err != nil {
			return err
		}
	}
	return nil
}


// --- WRITE METHODS

// writeOutPoint encodes op to the bitcoin protocol encoding for an OutPoint
// to w.
func writeOutPoint(w io.Writer, op *wire.OutPoint) error {
	_, err := w.Write(op.Hash[:])
	if err != nil {
		return err
	}

	return PutUint32(w, binary.LittleEndian, op.Index)
}

// writeTxIn encodes ti to the bitcoin protocol encoding for a transaction
// input (TxIn) to w.
func writeTxIn(w io.Writer, ti *wire.TxIn) error {
	err := writeOutPoint(w, &ti.PreviousOutPoint)
	if err != nil {
		return err
	}

	err = wire.WriteVarBytes(w, 0, ti.SignatureScript)
	if err != nil {
		return err
	}

	return PutUint32(w, binary.LittleEndian, ti.Sequence)
}

// WriteTxOut encodes to into the bitcoin protocol encoding for a transaction
// output (TxOut) to w.
func writeTxOut(w io.Writer, to *wire.TxOut, outType uint8) error {
   // write Tx type
   err := PutUint8(w, outType)
   if err != nil {
      return err
   }
   // write value
	err = PutUint64(w, binary.LittleEndian, uint64(to.Value))
	if err != nil {
		return err
	}

	return wire.WriteVarBytes(w, 0, to.PkScript)
}

// PutUint8 copies the provided uint8 into a buffer from the free list and
// writes the resulting byte to the given writer.
func PutUint8(w io.Writer, val uint8) error {
	buf := make([]byte, 1)
	buf[0] = val
	_, err := w.Write(buf)
	return err
}

// PutUint16 serializes the provided uint16 using the given byte order into a
// buffer from the free list and writes the resulting two bytes to the given
// writer.
func PutUint16(w io.Writer, byteOrder binary.ByteOrder, val uint16) error {
	buf := make([]byte, 2)
	byteOrder.PutUint16(buf, val)
	_, err := w.Write(buf)
	return err
}

// PutUint32 serializes the provided uint32 using the given byte order into a
// buffer from the free list and writes the resulting four bytes to the given
// writer.
func PutUint32(w io.Writer, byteOrder binary.ByteOrder, val uint32) error {
	buf := make([]byte, 4)
	byteOrder.PutUint32(buf, val)
	_, err := w.Write(buf)
	return err
}

// PutUint64 serializes the provided uint64 using the given byte order into a
// buffer from the free list and writes the resulting eight bytes to the given
// writer.
func PutUint64(w io.Writer, byteOrder binary.ByteOrder, val uint64) error {
	buf := make([]byte, 8)
	byteOrder.PutUint64(buf, val)
	_, err := w.Write(buf)
	return err
}

// WriteVarInt serializes val to w using a variable number of bytes depending
// on its value.
func WriteVarInt(w io.Writer, val uint64) error {
	if val < 0xfd {
		return PutUint8(w, uint8(val))
	}

	if val <= math.MaxUint16 {
		err := PutUint8(w, 0xfd)
		if err != nil {
			return err
		}
		return PutUint16(w, binary.LittleEndian, uint16(val))
	}

	if val <= math.MaxUint32 {
		err := PutUint8(w, 0xfe)
		if err != nil {
			return err
		}
		return PutUint32(w, binary.LittleEndian, uint32(val))
	}

	err := PutUint8(w, 0xff)
	if err != nil {
		return err
	}
	return PutUint64(w, binary.LittleEndian, val)
}

// writeElement writes the little endian representation of element to w.
func writeElement(w io.Writer, element interface{}) error {
	// Attempt to write the element based on the concrete type via fast
	// type assertions first.
	switch e := element.(type) {
	case int32:
		err := PutUint32(w, binary.LittleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil

	case uint32:
		err := PutUint32(w, binary.LittleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case int64:
		err := PutUint64(w, binary.LittleEndian, uint64(e))
		if err != nil {
			return err
		}
		return nil

	case uint64:
		err := PutUint64(w, binary.LittleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case bool:
		var err error
		if e {
			err = PutUint8(w, 0x01)
		} else {
			err = PutUint8(w, 0x00)
		}
		if err != nil {
			return err
		}
		return nil

	// Message header checksum.
	case [4]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	// IP address.
	case [16]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	case *chainhash.Hash:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil
	}

	// Fall back to the slower binary.Write if a fast path was not available
	// above.
	return binary.Write(w, binary.LittleEndian, element)
}

// writeElements writes multiple items to w.  It is equivalent to multiple
// calls to writeElement.
func writeElements(w io.Writer, elements ...interface{}) error {
	for _, element := range elements {
		err := writeElement(w, element)
		if err != nil {
			return err
		}
	}
	return nil
}

// baseSize returns the serialized size of the transaction without accounting
// for any witness data.
func baseSize(msg *wire.MsgTx) int {
	// Version 2 bytes + LockTime 4 bytes + flagWit 1 byte + Serialized varint size for the
	// number of transaction inputs and outputs.
	n := 7 + wire.VarIntSerializeSize(uint64(len(msg.TxIn))) +
		wire.VarIntSerializeSize(uint64(len(msg.TxOut)))

	for _, txIn := range msg.TxIn {
		n += txIn.SerializeSize()
	}

	for _, txOut := range msg.TxOut {
		n += txOut.SerializeSize()
	}

	return n
}

// --- ERRORS --- (from btcsuite/btcd/wire)
// MessageError describes an issue with a message.
// An example of some potential issues are messages from the wrong bitcoin
// network, invalid commands, mismatched checksums, and exceeding max payloads.
//
// This provides a mechanism for the caller to type assert the error to
// differentiate between general io errors such as io.EOF and issues that
// resulted from malformed messages.
type MessageError struct {
	Func        string // Function name
	Description string // Human readable description of the issue
}

// Error satisfies the error interface and prints human-readable errors.
func (e *MessageError) Error() string {
	if e.Func != "" {
		return fmt.Sprintf("%v: %v", e.Func, e.Description)
	}
	return e.Description
}

// messageError creates an error for the given function and description.
func messageError(f string, desc string) *MessageError {
	return &MessageError{Func: f, Description: desc}
}
