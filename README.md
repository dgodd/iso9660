# ISO9660 read cdrom iso
[![Build Status](https://travis-ci.org/dgodd/iso9660.svg?branch=master)](https://travis-ci.org/dgodd/iso9660)
[![Windows Build Status](https://ci.appveyor.com/api/projects/status/github/dgodd/iso9660?svg=true&branch=master)](https://ci.appveyor.com/project/dgodd/iso9660)

Read files from cdrom iso files, rockridge filenames are read.

### Example
```
subject, err = iso9660.New("./testdata/test.iso")
entries, err := subject.ReadDir("/dir/subdir")
b, err := subject.ReadFile("/long_dir_name/long_file_name.text")
```
