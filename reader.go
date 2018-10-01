package iso9660

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"

	mmap "github.com/edsrzf/mmap-go"
	"github.com/pkg/errors"
)

type Reader struct {
	fh *os.File
	m  mmap.MMap
	vp VolumePrimary
}

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

func New(path string) (*Reader, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "opening %s as mmaped iso file", path)
	}
	m, err := mmap.Map(fh, mmap.RDONLY, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "opening %s as mmaped iso file", path)
	}
	r := &Reader{fh: fh, m: m}
	if err := r.readVolumePrimary(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Reader) Close() error {
	e1 := r.m.Unmap()
	e2 := r.fh.Close()
	if e1 != nil {
		return e1
	}
	return e2
}

func (r *Reader) ReadDir(path string) ([]DirEntry, error) {
	var lbaStart, lbaLength int64
	if path == "/" || path == "\\" || path == "" {
		lbaStart = int64(r.vp.DirectoryEntryRoot.ExtentLocation) * 2048
		lbaLength = int64(r.vp.DirectoryEntryRoot.ExtentLength)
	} else {
		base := filepath.Base(path)
		dirs, err := r.ReadDir(filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		for _, dir := range dirs {
			if dir.ID == base {
				lbaStart = int64(dir.ExtentLocation) * 2048
				lbaLength = int64(dir.ExtentLength)
				break
			}
		}
	}
	if lbaLength == 0 {
		return nil, fmt.Errorf("dir not found: %s", path)
	}
	b := r.m[lbaStart:(lbaStart + lbaLength)]
	var dirEntry DirEntry
	var dirs []DirEntry
	var offset int64
	for {
		if offset >= int64(r.vp.DirectoryEntryRoot.ExtentLength) {
			break
		}
		parseDirEntry(b[offset:], &dirEntry)
		if dirEntry.Length == 0 {
			break
		}
		dirs = append(dirs, dirEntry)
		offset += int64(dirEntry.Length)
	}

	return dirs, nil
}

func (r *Reader) ReadFile(path string) ([]byte, error) {
	entry, err := r.entry(path)
	if err != nil {
		return nil, errors.Wrap(err, "ReadFile")
	}
	offset := entry.ExtentLocation * 2048
	length := entry.ExtentLength
	return r.m[offset:(offset + length)], nil
}

func (r *Reader) entry(path string) (DirEntry, error) {
	base := filepath.Base(path)
	entries, err := r.ReadDir(filepath.Dir(path))
	if err != nil {
		return DirEntry{}, err
	}
	for _, e := range entries {
		if e.ID == base {
			return e, nil
		}
	}
	return DirEntry{}, fmt.Errorf("not found: %s", path)
}

func (r *Reader) readVolumePrimary() error {
	b := r.m[32768:(32768 + 2048)]
	if b[0] == 255 {
		return errors.New("no volumes found in iso")
	}
	if b[0] != 1 {
		return fmt.Errorf("Expected Volume type of 1 (primary): %d", b[0])
	}
	if b[6] != 1 {
		return fmt.Errorf("Expected Volume version of 1: %d", b[6])
	}
	if string(b[1:6]) != "CD001" {
		return fmt.Errorf("Expected Volume ID of CD001: %s", string(b[1:6]))
	}

	b = b[7:]
	r.vp.SystemIdentifier = string(b[1:33])
	r.vp.VolumeIdentifier = string(b[33:64])
	r.vp.VolumeSpaceSize = binary.LittleEndian.Uint32(b[73:77])
	r.vp.VolumeSetSize = binary.LittleEndian.Uint16(b[113:115])
	r.vp.VolumeSequenceNumber = binary.LittleEndian.Uint16(b[117:119])
	r.vp.LogicalBlockSize = binary.LittleEndian.Uint16(b[121:123])
	r.vp.PathTableSize = binary.LittleEndian.Uint32(b[125:129])
	r.vp.LocationTypePathTable = binary.LittleEndian.Uint32(b[133:137])
	r.vp.LocationOptionalTypePathTable = binary.LittleEndian.Uint32(b[137:141])
	parseDirEntry(b[149:183], &r.vp.DirectoryEntryRoot)
	r.vp.VolumeSetIdentifier = string(b[183:(183 + 128)])
	r.vp.PublisherIdentifier = string(b[311:(311 + 128)])
	r.vp.DataPreparerIdentifier = string(b[439:(439 + 128)])
	r.vp.ApplicationIdentifier = string(b[567:(567 + 128)])
	r.vp.CopyrightFileIdentifier = string(b[695:(695 + 38)])
	r.vp.AbstractFileIdentifier = string(b[737:(737 + 36)])
	r.vp.BibliographicFileIdentifier = string(b[769:(769 + 37)])
	r.vp.FileStructureVersion = b[874]

	// fmt.Printf("SystemIdentifier: %s\n", r.vp.SystemIdentifier)
	// fmt.Printf("VolumeIdentifier: %s\n", r.vp.VolumeIdentifier)
	// fmt.Printf("VolumeSpaceSize: %d\n", r.vp.VolumeSpaceSize)
	// fmt.Printf("VolumeSetSize: %d\n", r.vp.VolumeSetSize)
	// fmt.Printf("VolumeSequenceNumber: %d\n", r.vp.VolumeSequenceNumber)
	// fmt.Printf("LogicalBlockSize: %d\n", r.vp.LogicalBlockSize)
	// fmt.Printf("PathTableSize: %d\n", r.vp.PathTableSize)
	// fmt.Printf("LocationTypePathTable: %d\n", r.vp.LocationTypePathTable)
	// fmt.Printf("LocationOptionalTypePathTable: %d\n", r.vp.LocationOptionalTypePathTable)
	// fmt.Printf("DirectoryEntryRoot: %#v\n", r.vp.DirectoryEntryRoot)
	// fmt.Printf("VolumeSetIdentifier: '%s'\n", r.vp.VolumeSetIdentifier)
	// fmt.Printf("PublisherIdentifier: '%s'\n", r.vp.PublisherIdentifier)
	// fmt.Printf("DataPreparerIdentifier: '%s'\n", r.vp.DataPreparerIdentifier)
	// fmt.Printf("ApplicationIdentifier: '%s'\n", r.vp.ApplicationIdentifier)
	// fmt.Printf("CopyrightFileIdentifier: '%s'\n", r.vp.CopyrightFileIdentifier)
	// fmt.Printf("AbstractFileIdentifier: '%s'\n", r.vp.AbstractFileIdentifier)
	// fmt.Printf("BibliographicFileIdentifier: '%s'\n", r.vp.BibliographicFileIdentifier)
	// // fmt.Printf("VolumeCreationDateandTime: %v\n", r.vp.VolumeCreationDateandTime)
	// // fmt.Printf("VolumeModificationDateTime: %v\n", r.vp.VolumeModificationDateTime)
	// // fmt.Printf("VolumeExpirationDateTime: %v\n", r.vp.VolumeExpirationDateTime)
	// // fmt.Printf("VolumeEffectiveDateTime: %v\n", r.vp.VolumeEffectiveDateTime)
	// fmt.Printf("FileStructureVersion: %v\n", r.vp.FileStructureVersion)

	if r.m[32768+2048] != 255 {
		return errors.New("expected to only find one volume")
	}
	return nil
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

	if len(e.ID) > 2 && e.ID[len(e.ID)-2] == ';' {
		// fmt.Println("EID:", e.ID)
		e.ID = e.ID[:(len(e.ID) - 2)]
	}

	offset := 33 + e.IDLength
	if e.IDLength%2 == 0 {
		offset++
	}
	if offset < e.Length {
		// fmt.Printf("E: %#v\n", e)
		// fmt.Printf("B: %v\n", b[offset:(e.Length-offset)])
		// fmt.Printf("S: %s\n", b[offset:(e.Length-offset)])
		rr, err := parseSUSP(b[offset:(e.Length - offset)])
		if err == nil && rr.Name != "" {
			e.ID = rr.Name
		}
	}
}

type RR struct {
	Name string
}

func parseSUSP(b []byte) (RR, error) {
	offset := int64(0)
	var rr RR
	for {
		if offset >= int64(len(b)) {
			break
		}
		sig := string(b[offset:(offset + 2)])
		len := b[offset+2]
		if b[offset+3] != 1 {
			return RR{}, fmt.Errorf("expected version == 1 in SUSP record: %d", b[offset+3])
		}
		switch sig {
		case "RR":
			// no-op
		case "NM":
			rr.Name = string(b[(offset + 5):(offset + int64(len))])
		default:
			// fmt.Println(string(sig), len)
			// fmt.Println(b[offset:(offset + int64(len))])
		}
		offset += int64(len)
	}
	return rr, nil
}
