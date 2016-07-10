package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"
)

type Header struct {
	MagicString [4]byte
	Version     uint8
	NumRecords  uint32
}

type RecordTypeEnum int8

const (
	Debit        RecordTypeEnum = 0x00
	Credit       RecordTypeEnum = 0x01
	StartAutopay RecordTypeEnum = 0x02
	EndAutopay   RecordTypeEnum = 0x03
	MagicString                 = "MPS7"
)

func (rte RecordTypeEnum) String() string {
	switch rte {
	case Credit:
		return "Credit"
	case Debit:
		return "Debit"
	case StartAutopay:
		return "Start AutoPay"
	case EndAutopay:
		return "End AutoPay"
	}
	return "Unknown record type"
}

type BaseRecord struct {
	RecordType RecordTypeEnum
	Timestamp  uint32
	UserID     uint64
}

type TxnRecord struct {
	base         BaseRecord
	dollarAmount float64
}

type FileData struct {
	header  Header
	records []TxnRecord
}

func (fd *FileData) Load(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Printf("error %v\n", err)
		return
	}
	binary.Read(file, binary.BigEndian, &fd.header)
	if fd.header.Validate() {
		fd.records = make([]TxnRecord, fd.header.NumRecords)
		for i := uint32(0); i < fd.header.NumRecords; i++ {
			binary.Read(file, binary.BigEndian, &fd.records[i].base)
			if fd.records[i].base.RecordType == Credit || fd.records[i].base.RecordType == Debit {
				//Get the amount for credit/debit
				binary.Read(file, binary.BigEndian, &fd.records[i].dollarAmount)
			}
		}
		//This should be the end of the file
	} else {
		fmt.Printf("Error Magic string mismatch: %v\n", fd.header.MagicString)
	}
	file.Close()
}

func (txr TxnRecord) String() string {
	return fmt.Sprintf("%v\t%v\t%d\t%f", time.Unix(int64(txr.base.Timestamp), 0), txr.base.RecordType, txr.base.UserID, txr.dollarAmount)
}

func (h Header) Validate() bool {
	//Test Magic string
	return string(h.MagicString[:]) == MagicString
}

//Parse the txnlog.dat
func main() {
	//Open the file
	var txnData FileData
	txnData.Load("txnlog.dat")

	totalDebits, totalCredits, apStarts, apEnds, userBal := 0.0, 0.0, 0, 0, 0.0
	for _, txn := range txnData.records {
		fmt.Println(txn)
		if txn.base.RecordType == Debit {
			totalDebits += txn.dollarAmount
		}
		if txn.base.RecordType == Credit {
			totalCredits += txn.dollarAmount
		}
		if txn.base.RecordType == StartAutopay {
			apStarts++
		}
		if txn.base.RecordType == EndAutopay {
			apEnds++
		}
		if txn.base.UserID == 2456938384156277127 {
			if txn.base.RecordType == Credit {
				userBal += txn.dollarAmount
			}
			if txn.base.RecordType == Debit {
				userBal -= txn.dollarAmount
			}
		}
	}
	fmt.Println("What is the total amount in dollars of debits? ", totalDebits)

	fmt.Println("What is the total amount in dollars of credits?", totalCredits)
	fmt.Println("How many autopays were started?", apStarts)
	fmt.Println("How many autopays were ended?", apEnds)
	fmt.Println("What is balance of user ID 2456938384156277127?", userBal)

}
