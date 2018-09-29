package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"golang.org/x/exp/mmap"
)

type VolumePrimary struct {
	Type                          byte
	ID                            [5]byte
	Version                       byte
	SystemIdentifier              string
	VolumeIdentifier              string
	VolumeSpaceSize               uint32
	VolumeSetSize                 uint16
	VolumeSequenceNumber          uint16 // [4]byte
	LogicalBlockSize              uint16 // [4]byte
	PathTableSize                 uint32 // [8]byte
	LocationTypePathTable         uint32 // [4]byte
	LocationOptionalTypePathTable uint32 // [4]byte
	DirectoryEntryRoot            DirEntry
	VolumeSetIdentifier           string
	PublisherIdentifier           string
	DataPreparerIdentifier        string
	ApplicationIdentifier         string
	CopyrightFileIdentifier       string
	AbstractFileIdentifier        string
	BibliographicFileIdentifier   string
	// VolumeCreationDateandTime     [17]byte
	// VolumeModificationDateTime    [17]byte
	// VolumeExpirationDateTime      [17]byte
	// VolumeEffectiveDateTime       [17]byte
	FileStructureVersion byte // uint8 // byte
}

type DirEntry struct {
	Length         byte      // 0	1	Length of Directory Record.
	EARLength      byte      // 1	1	Extended Attribute Record length.
	ExtentLocation uint32    // 2	8	Location of extent (LBA) in both-endian format.
	ExtentLength   uint32    // 10	8	Data length (size of extent) in both-endian format.
	RecordingDate  time.Time // 18	7	Recording date and time (see format below).
	// 25	1	File flags (see below).
	// 26	1	File unit size for files recorded in interleaved mode, zero otherwise.
	// 27	1	Interleave gap size for files recorded in interleaved mode, zero otherwise.
	// 28	4	Volume sequence number - the volume that this extent is recorded on, in 16 bit both-endian format.
	IDLength byte   // 32	1	Length of file identifier (file name). This terminates with a ';' character followed by the file ID number in ASCII coded decimal ('1').
	ID       string // 33	(variable)	File identifier.
	// (variable)	1	Padding field - zero if length of file identifier is even, otherwise, this field is not present. This means that a directory entry will always start on an even byte number.
	// (variable)	(variable) System Use - The remaining bytes up to the maximum record size of 255 may be used for extensions of ISO 9660. The most common one is the System Use Share Protocol (SUSP) and its application, the Rock Ridge Interchange Protocol (RRIP).
}

func parseTime(b []byte) time.Time {
	// offset = b[6] // Offset from GMT in 15 minute intervals from -48 (West) to +52 (East)
	return time.Date(1900+int(b[0]), time.Month(b[1]), int(b[2]), int(b[3]), int(b[4]), int(b[5]), 0, &time.Location{})
}

func parseDirEntry(b []byte, e *DirEntry) {
	e.Length = b[0]
	e.EARLength = b[1]
	e.ExtentLocation = binary.LittleEndian.Uint32(b[2:6])
	e.ExtentLength = binary.LittleEndian.Uint32(b[10:14])
	e.RecordingDate = parseTime(b[18:25])
	e.IDLength = b[32]
	e.ID = string(b[33:(33 + e.IDLength)])
}

func main() {
	// f, err := os.Open("/home/dgodd/Downloads/cf-deps-0.107.0.iso")
	// f, err := os.Open("/home/dgodd/Downloads/test.iso")
	f, err := mmap.Open("./testdata/test.iso")
	if err != nil {
		panic(err)
		// log.Fatal(err)
	}

	fmt.Println("FILE LEN:", f.Len())
	volumeOffset := int64(32768)

	var volumes []VolumePrimary
	for {
		b := make([]byte, 2048)
		if _, err := f.ReadAt(b, volumeOffset); err != nil {
			// log.Fatal(err)
			panic(err)
		}
		if b[0] == 255 {
			break
		}
		var v VolumePrimary
		if b[0] != 1 {
			log.Fatalf("Expected Volume type of 1 (primary): %d", b[0])
		}
		if b[6] != 1 {
			log.Fatalf("Expected Volume version of 1: %d", b[6])
		}
		if string(b[1:6]) != "CD001" {
			log.Fatalf("Expected Volume ID of CD001: %s", string(b[1:6]))
		}

		b = b[7:]
		v.SystemIdentifier = string(b[1:33])
		v.VolumeIdentifier = string(b[33:64])
		v.VolumeSpaceSize = binary.LittleEndian.Uint32(b[73:77])
		v.VolumeSetSize = binary.LittleEndian.Uint16(b[113:115])
		v.VolumeSequenceNumber = binary.LittleEndian.Uint16(b[117:119])
		v.LogicalBlockSize = binary.LittleEndian.Uint16(b[121:123])
		v.PathTableSize = binary.LittleEndian.Uint32(b[125:129])
		v.LocationTypePathTable = binary.LittleEndian.Uint32(b[133:137])
		v.LocationOptionalTypePathTable = binary.LittleEndian.Uint32(b[137:141])
		parseDirEntry(b[149:183], &v.DirectoryEntryRoot)
		v.VolumeSetIdentifier = string(b[183:(183 + 128)])
		v.PublisherIdentifier = string(b[311:(311 + 128)])
		v.DataPreparerIdentifier = string(b[439:(439 + 128)])
		v.ApplicationIdentifier = string(b[567:(567 + 128)])
		v.CopyrightFileIdentifier = string(b[695:(695 + 38)])
		v.AbstractFileIdentifier = string(b[737:(737 + 36)])
		v.BibliographicFileIdentifier = string(b[769:(769 + 37)])
		v.FileStructureVersion = b[874]

		fmt.Printf("SystemIdentifier: %s\n", v.SystemIdentifier)
		fmt.Printf("VolumeIdentifier: %s\n", v.VolumeIdentifier)
		fmt.Printf("VolumeSpaceSize: %d\n", v.VolumeSpaceSize)
		fmt.Printf("VolumeSetSize: %d\n", v.VolumeSetSize)
		fmt.Printf("VolumeSequenceNumber: %d\n", v.VolumeSequenceNumber)
		fmt.Printf("LogicalBlockSize: %d\n", v.LogicalBlockSize)
		fmt.Printf("PathTableSize: %d\n", v.PathTableSize)
		fmt.Printf("LocationTypePathTable: %d\n", v.LocationTypePathTable)
		fmt.Printf("LocationOptionalTypePathTable: %d\n", v.LocationOptionalTypePathTable)
		fmt.Printf("DirectoryEntryRoot: %#v\n", v.DirectoryEntryRoot)
		fmt.Printf("VolumeSetIdentifier: '%s'\n", v.VolumeSetIdentifier)
		fmt.Printf("PublisherIdentifier: '%s'\n", v.PublisherIdentifier)
		fmt.Printf("DataPreparerIdentifier: '%s'\n", v.DataPreparerIdentifier)
		fmt.Printf("ApplicationIdentifier: '%s'\n", v.ApplicationIdentifier)
		fmt.Printf("CopyrightFileIdentifier: '%s'\n", v.CopyrightFileIdentifier)
		fmt.Printf("AbstractFileIdentifier: '%s'\n", v.AbstractFileIdentifier)
		fmt.Printf("BibliographicFileIdentifier: '%s'\n", v.BibliographicFileIdentifier)
		// fmt.Printf("VolumeCreationDateandTime: %v\n", v.VolumeCreationDateandTime)
		// fmt.Printf("VolumeModificationDateTime: %v\n", v.VolumeModificationDateTime)
		// fmt.Printf("VolumeExpirationDateTime: %v\n", v.VolumeExpirationDateTime)
		// fmt.Printf("VolumeEffectiveDateTime: %v\n", v.VolumeEffectiveDateTime)
		fmt.Printf("FileStructureVersion: %v\n", v.FileStructureVersion)

		volumes = append(volumes, v)
		volumeOffset += 2048
	}

	if len(volumes) != 1 {
		log.Fatalf("Expected 1 Volume: %d", len(volumes))
	}
	volume := volumes[0]

	b := make([]byte, volume.DirectoryEntryRoot.ExtentLength)
	if _, err := f.ReadAt(b, int64(volume.DirectoryEntryRoot.ExtentLocation*2048)); err != nil {
		// log.Fatal(err)
		panic(err)
	}
	offset := int64(0)
	var dirEntry DirEntry
	for {
		parseDirEntry(b[offset:], &dirEntry)
		if dirEntry.Length == 0 {
			break
		}
		fmt.Printf("DIR: %#v\n", dirEntry)
		offset += int64(dirEntry.Length)
	}

	b = make([]byte, volume.PathTableSize)
	if _, err := f.ReadAt(b, int64(volume.LocationTypePathTable*2048)); err != nil {
		// log.Fatal(err)
		panic(err)
	}
	fmt.Println("B:", string(b))
	offset = int64(0)
	for {
		if offset >= int64(volume.PathTableSize) || b[offset] == 0 {
			break
		}
		length := b[offset]
		earLength := b[offset+1]
		lba := binary.LittleEndian.Uint32(b[(offset + 2):(offset + 6)])
		parent := binary.LittleEndian.Uint16(b[(offset + 6):(offset + 8)])
		name := string(b[(offset + 8):(offset + 8 + int64(length))])
		// fmt.Printf("DIR: %#v\n", dirEntry)
		fmt.Println("PT:", length, earLength, lba, parent, name)
		offset += 8 + int64(length)
		if length%2 == 1 {
			offset++
		}
	}
}

// ./file1.txt
// ./dir2
// ./dir2/file3.txt
// ./dir2/dir3
// ./dir2/dir3/file4.txt
// ./dir2/dir3/file5.txt
// ./dir2/long_file_name.txt
// ./dir1
// ./dir1/file2.txt
