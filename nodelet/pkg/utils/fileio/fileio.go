package fileio

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Pf9FileIO : A Platform9 wrapper for doing file IO operations
type Pf9FileIO struct {
	log *zap.SugaredLogger
}

// New returns new instance of Pf9FileIO for file IO ops
func New() FileInterface {
	return &Pf9FileIO{
		log: zap.S(),
	}
}

// FileInterface interface contains ways to R/W data From/To a file respectively
type FileInterface interface {
	TouchFile(string) error
	GetFileInfo(string) (os.FileInfo, error)
	RenameAndMoveFile(string, string) error
	CopyFile(string, string) error
	DeleteFile(string) error
	ReadFile(string) ([]byte, error)
	ReadFileByLine(string) ([]string, error)
	WriteToFile(string, interface{}, bool) error
	ReadJSONFile(string, interface{}) error
	ListFiles(string) ([]string, error)
	GenerateChecksum(string) error
	VerifyChecksum(string) (bool, error)
	GenerateHashForDir(string) ([]string, error)
	GenerateHashForFile(string) (string, error)
	NewYamlFromTemplateYaml(string, string, interface{}) error
	ListFilesWithPatterns(string, []string) ([]string, error)
}

// TouchFile creates an empty file
func (f *Pf9FileIO) TouchFile(filename string) error {
	touchFile, err := os.Create(filename)

	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return err
	}

	defer touchFile.Close()

	return nil
}

// GetFileInfo fetches file details
func (f *Pf9FileIO) GetFileInfo(filename string) (os.FileInfo, error) {
	info, err := os.Stat(filename)
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return nil, err
	}

	f.log.Infof("File Details: %v", info)
	return info, nil
}

// RenameAndMoveFile renames and/or moves file
func (f *Pf9FileIO) RenameAndMoveFile(originalFile, newFile string) error {
	err := os.Rename(originalFile, newFile)
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return err
	}
	return nil
}

// CopyFile copies a file
func (f *Pf9FileIO) CopyFile(originalFile, duplicateFile string) error {
	origFile, err := os.Open(originalFile)
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return err
	}
	defer origFile.Close()

	dupFile, err := os.Create(duplicateFile)
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return err
	}
	defer dupFile.Close()

	_, err = io.Copy(dupFile, origFile)
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return err
	}

	err = dupFile.Sync()
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return err
	}
	return nil
}

// DeleteFile deletes a file
func (f *Pf9FileIO) DeleteFile(filename string) error {
	if err := os.Remove(filename); err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return err
	}
	return nil
}

// ReadFile reads an entire file and returns as bytes array
//
// Should avoid reading large files
func (f *Pf9FileIO) ReadFile(filename string) ([]byte, error) {
	var data []byte
	file, err := os.Open(filename)
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return []byte{}, err
	}
	defer file.Close()

	data, err = ioutil.ReadAll(file)
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return []byte{}, err
	}
	return data, nil
}

// ReadFileByLine reads a file line by line and returns it as slice
//
// Should be avoided when reading large files
func (f *Pf9FileIO) ReadFileByLine(filename string) ([]string, error) {
	var err error
	fileContents := []string{}

	file, err := os.OpenFile(filename, os.O_RDONLY, 0666)
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return fileContents, err
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		fileContents = append(fileContents, line)
	}

	return fileContents, err
}

// WriteToFile writes data to file
func (f *Pf9FileIO) WriteToFile(filename string, data interface{}, append bool) error {
	var (
		err  error
		file *os.File
		flag int
	)

	if append {
		flag = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	} else {
		flag = os.O_WRONLY | os.O_TRUNC | os.O_CREATE
	}

	file, err = os.OpenFile(filename, flag, 0666)
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return err
	}
	defer file.Close()

	switch data.(type) {
	case []byte:
		_, err := file.Write(data.([]byte))
		if err != nil {
			f.log.Errorf("Error: %s", err.Error())
			return err
		}
	case []string:
		for _, str := range data.([]string) {
			_, err := file.Write([]byte(str + "\n"))
			if err != nil {
				f.log.Errorf("Error: %s", err.Error())
				return err
			}
		}
	case string:
		_, err := file.Write([]byte(data.(string)))
		if err != nil {
			f.log.Errorf("Error: %s", err.Error())
			return err
		}
	default:
		err = fmt.Errorf("invalid data provided to write to a file. Provide []byte, []string or string")
		f.log.Errorf("Error: %v", err)
		return err
	}
	return nil
}

// ReadJSONFile reads a JSON file and updates the map passed as argument
func (f *Pf9FileIO) ReadJSONFile(filename string, output interface{}) error {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0666)
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return err
	}

	err = json.Unmarshal(data, &output)
	if err != nil {
		f.log.Errorf("Error: %s", err.Error())
		return err
	}

	return nil
}

// ListFiles lists all the files in the directory
func (f *Pf9FileIO) ListFiles(dirname string) ([]string, error) {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		f.log.Errorf("Could not read files in directory: %s", dirname)
		return nil, err
	}
	var filenames []string = make([]string, len(files))
	for i, file := range files {
		filenames[i] = file.Name()
	}
	return filenames, nil
}

// GenerateChecksum generates sha256 checksum for files and writes the checksum file in checksum sub-dir in given directory
func (f *Pf9FileIO) GenerateChecksum(imageDir string) error {

	hash, err := f.GenerateHashForDir(imageDir)
	if err != nil {
		return errors.Wrapf(err, "could not generate hash for: %s", imageDir)
	}
	ChecksumDir := fmt.Sprintf("%s/checksum", imageDir)
	if _, err := os.Stat(ChecksumDir); os.IsNotExist(err) {
		if err := os.MkdirAll(ChecksumDir, os.ModePerm); err != nil {
			return errors.Wrapf(err, "failed to create directory: %s", ChecksumDir)
		}
	}
	ChecksumFile := fmt.Sprintf("%s/sha256sums.txt", ChecksumDir)
	err = f.WriteToFile(ChecksumFile, hash, false)
	if err != nil {
		return err
	}
	return nil
}

// VerifyChecksum verifies the current sha256 checksum of all files with checksum file
func (f *Pf9FileIO) VerifyChecksum(imageDir string) (bool, error) {

	currentHash, err := f.GenerateHashForDir(imageDir)
	if err != nil {
		return false, errors.Wrapf(err, "could not generate hash for: %s", imageDir)
	}
	checksumFile := fmt.Sprintf("%s/checksum/sha256sums.txt", imageDir)
	prevHash, err := f.ReadFileByLine(checksumFile)
	if err != nil {
		return false, err
	}
	res := stringSlicesEqual(currentHash, prevHash)
	if res {
		return true, nil
	} else {
		err = f.WriteToFile(checksumFile, currentHash, false)
		if err != nil {
			return false, err
		}
		return false, nil
	}
}

// GenerateHashForDir generates sha256 hash for files in directory and returns string slice of hashes of files
func (f *Pf9FileIO) GenerateHashForDir(imageDir string) ([]string, error) {
	data := []string{}
	items, _ := ioutil.ReadDir(imageDir)
	for _, item := range items {
		if item.IsDir() {
			continue
		} else {
			imageFile := fmt.Sprintf("%s/%s", imageDir, item.Name())
			fileData, err := f.GenerateHashForFile(imageFile)
			if err != nil {
				return nil, err
			}
			data = append(data, fileData)
		}
	}
	return data, nil
}

// GenerateHashForFile generates sha256 hash for given file and returns string of hash of file
func (f *Pf9FileIO) GenerateHashForFile(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	data := h.Sum(nil)
	fileData := hex.EncodeToString(data)
	fileData = fmt.Sprintf("%s  %s", fileData, fileName)
	return fileData, nil
}

// NewYamlFromTemplateYaml creates new yaml from template yaml file by replacing the value in provided data
func (f *Pf9FileIO) NewYamlFromTemplateYaml(templFile string, outFile string, data interface{}) error {

	fw, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer fw.Close()
	t, err := template.ParseFiles(templFile)
	if err != nil {
		return err
	}
	err = t.Execute(fw, data)
	if err != nil {
		return fmt.Errorf("error executing template: %s", err)
	}
	return nil
}

// ListfilesWithPattern returns list of filenames with given patterns from given directory *recursively*
func (f *Pf9FileIO) ListFilesWithPatterns(root string, patterns []string) ([]string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		for _, p := range patterns {

			if matched, err := filepath.Match(p, filepath.Base(path)); err != nil {
				return err
			} else if matched {
				matches = append(matches, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

// stringSlicesEqual states whether two string slices are equal or not
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
