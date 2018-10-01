# ISO9660 read cdrom iso
[![Build Status](https://travis-ci.org/dgodd/iso9660.svg?branch=master)](https://travis-ci.org/dgodd/iso9660)

Read files from cdrom iso files.

### Example
```
subject, err = iso9660.New("./testdata/test.iso")
entries, err := subject.ReadDir("/dir/subdir")
b, err := subject.ReadFile("/dir/file.txt")
```
