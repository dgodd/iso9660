package iso9660_test

import (
	"testing"

	"github.com/dgodd/iso9660"
	"github.com/google/go-cmp/cmp"
	"github.com/sclevine/spec"
)

func TestISO9660(t *testing.T) {
	spec.Run(t, "iso9660", testISO9660)
}

func testISO9660(t *testing.T, when spec.G, it spec.S) {
	when("#ReadDir", func() {
		it("reads the root dir", func() {
			subject, err := iso9660.New("./testdata/test.iso")
			assertNil(t, err)

			entries, err := subject.ReadDir("/")
			assertNil(t, err)

			names := make([]string, len(entries))
			for i, e := range entries {
				names[i] = e.ID
			}

			assertEq(t, names, []string{"\x00", "\x01", "dir1", "dir2", "file1.txt"})
		})
		it.Focus("reads sub dirs", func() {
			subject, err := iso9660.New("./testdata/test.iso")
			assertNil(t, err)

			entries, err := subject.ReadDir("/DIR2")
			assertNil(t, err)

			names := make([]string, len(entries))
			for i, e := range entries {
				names[i] = e.ID
			}

			assertEq(t, names, []string{"\x00", "\x01", "dir3", "file3.txt", "long_file_name.txt"})
		})
		it.Focus("reads sub dirs with long names", func() {
			subject, err := iso9660.New("./testdata/test.iso")
			assertNil(t, err)

			entries, err := subject.ReadDir("/long_dir_name/long_sub_dir_name")
			assertNil(t, err)

			names := make([]string, len(entries))
			for i, e := range entries {
				names[i] = e.ID
			}

			assertEq(t, names, []string{"\x00", "\x01", "long_file_name_2.txt"})
		})
	})
	when("#AllDirs", func() {
		when("valid iso file", func() {
			it("returns all of the dirs", func() {
				subject, err := iso9660.New("./testdata/test.iso")
				assertNil(t, err)

				dirs := subject.AllDirs()
				assertEq(t, dirs, []string{"/", "/DIR1", "/DIR2", "/DIR2/DIR3"})
			})
		})
	})
}

func assertEq(t *testing.T, actual, expected interface{}) {
	t.Helper()
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Fatal(diff)
	}
}

func assertNil(t *testing.T, actual interface{}) {
	t.Helper()
	if actual != nil {
		t.Fatalf("Expected nil: %s", actual)
	}
}
