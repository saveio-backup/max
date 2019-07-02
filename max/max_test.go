package max

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	ipld "gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/saveio/max/importer/helpers"
	"github.com/saveio/max/max/fsstore"
	ml "github.com/saveio/max/merkledag"
	sdk "github.com/saveio/themis-go-sdk"
)

const CHUNK_SIZE = 256 * 1024
const (
	GC_PERIOD           = "1h"
	GC_PERIOD_IMMEDIATE = "0s"
	GC_PERIOD_TEST      = "5s"
)

func makeTempFile(dir string, data []byte) (string, error) {
	f, err := ioutil.TempFile(dir, "file")
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}

func makeFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0766)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return err
	}
	return nil
}

// the cids contains all the keys of blocks for a file, buf contain file content
func checkFileContent(max *MaxService, cids []*cid.Cid, buf []byte) error {
	var index int
	for _, c := range cids {
		blk, err := max.GetBlock(c)
		if err != nil {
			return err
		}

		// check if ipld node, data retrieved from filestore is file content wrapped by basicblock
		dagNode, err := ml.DecodeProtobufBlock(blk)
		if err == nil {
			if len(dagNode.Links()) == 0 {
				return errors.New("ipld node with no link")
			}
			continue
		}

		start := index * CHUNK_SIZE
		end := (index + 1) * CHUNK_SIZE
		if end > len(buf) {
			end = len(buf)
		}

		if _, ok := blk.(ml.RawNode); ok {
			if !bytes.Equal(blk.RawData(), buf[start:end]) {
				return errors.New("data didnot match on the way out")
			}
		}
		index++
	}

	return nil
}

//NOTE: if some cid is shared with other file which is not deleted yet, it will return error
func checkFileBlocksNoExist(max *MaxService, cids []*cid.Cid) error {
	for _, c := range cids {
		_, err := max.GetBlock(c)
		if err == nil {
			return errors.New("file block exist")
		}
	}
	return nil
}

func getCidsFromNodelist(nodeList []*helpers.UnixfsNode) ([]*cid.Cid, error) {
	var cids []*cid.Cid

	for _, node := range nodeList {
		dagNode, err := node.GetDagNode()
		if err != nil {
			return nil, err
		}
		cids = append(cids, dagNode.Cid())
	}

	return cids, nil
}

func getCidsFromNodelistForRawNodes(nodeList []*helpers.UnixfsNode) ([]*cid.Cid, error) {
	var cids []*cid.Cid

	for _, node := range nodeList {
		dagNode, err := node.GetDagNode()
		if err != nil {
			return nil, err
		}
		if len(dagNode.Links()) == 0 {
			cids = append(cids, dagNode.Cid())
		}
	}

	return cids, nil
}

func getBufWithPrefix(buf []byte, prefix string) []byte {
	bufWithPrefix := []byte(prefix)
	bufWithPrefix = append(bufWithPrefix, buf...)

	return bufWithPrefix
}

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func checkFileContentWithOffset(max *MaxService, cids []*cid.Cid, offsets []uint64, buf []byte) error {
	var index int
	for i, c := range cids {
		blk, err := max.GetBlock(c)
		if err != nil {
			return err
		}

		// all nodes should be raw nodes
		_, err = ml.DecodeProtobufBlock(blk)
		if err == nil {
			return errors.New("all nodes should be rawnodes")
		}

		start := offsets[i]
		end := start + CHUNK_SIZE
		if end > uint64(len(buf)) {
			end = uint64(len(buf))
		}

		if !bytes.Equal(blk.RawData(), buf[start:end]) {
			return errors.New("data didnot match on the way out")
		}
		index++
	}

	return nil
}

type Config struct {
	repoRoot   string
	createTmp  bool // if true create tmp dir and use as the repoRoot , if false, repoRoot should be used for the api
	fsType     FSType
	chunkSize  uint64
	gcPeriod   string
	chain      *sdk.Chain
	maxStorage string
}

var repoPaths = []string{
	".ont-ipfs",
	"blocks",
	"datastore",
	"keystore",
	"config",
	"datastore_spec",
	"version",
}

func initFsFromConfig(config *Config) (max *MaxService, repoRoot string, err error) {
	repoRoot = config.repoRoot

	if config.createTmp {
		repoRoot, err = ioutil.TempDir("", "max-test")
		if err != nil {
			return nil, "", err
		}
	}

	fsConfig := &FSConfig{
		RepoRoot:   repoRoot,
		FsType:     config.fsType,
		ChunkSize:  config.chunkSize,
		GcPeriod:   config.gcPeriod,
		MaxStorage: config.maxStorage,
	}

	max, err = NewMaxService(fsConfig, config.chain)
	if err != nil {
		return nil, "", err
	}

	return max, repoRoot, err
}
func TestNewMaxServiceParams(t *testing.T) {
	var services []*MaxService

	defer func() {
		// close the repo in order to delete files
		for _, max := range services {
			max.repo.Close()
		}

		for _, path := range repoPaths {
			os.RemoveAll("./" + path)
		}

		os.RemoveAll("./tmp")
	}()

	cases := map[*Config]bool{
		// normal case
		&Config{"", true, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}: true,

		//repoRoot tests
		&Config{"", false, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}:                                   false,
		&Config{"./", false, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}:                                 true,
		&Config{"/", false, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}:                                  false,
		&Config{"./tmp", false, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}:                              true,
		&Config{getCurrentDirectory() + "/tmp2/tmp3", false, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}: true,

		// fsType tests
		&Config{"", true, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}:  true,
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}: true,
		&Config{"", true, 2, CHUNK_SIZE, GC_PERIOD, nil, ""}:             false,
		&Config{"", true, 100, CHUNK_SIZE, GC_PERIOD, nil, ""}:           false,

		// fsRoot, chunksize are used by filestore and will not be checked in MaxService
		// gcPeriod tests, gc only applys to blockstore
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, "0", nil, ""}:   true,
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, "0s", nil, ""}:  true,
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, "10s", nil, ""}: true,
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, "10m", nil, ""}: true,
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, "10h", nil, ""}: true,

		// maxStorage test
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, "0"}:    false,
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}:     true,
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, "100M"}: true,
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, "100S"}: false,
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, "10G"}:  true,
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, "10T"}:  true,
		&Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, "10a"}:  false,
	}

	os.Mkdir("./tmp", 777)

	for config, expected := range cases {

		max, _, err := initFsFromConfig(config)
		if err == nil && max != nil {
			services = append(services, max)
		}

		result := (err == nil)
		if result != expected {
			t.Fatalf("failed to call API NewMaxService with config : %v\n, expected : %v, err : %s", config, expected, err)
		}
	}
}

func TestNewMaxServiceRepoInit(t *testing.T) {
	testdir, err := ioutil.TempDir("", "filestore-test")
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewMaxService(&FSConfig{testdir, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewMaxServiceRepoInitAlready(t *testing.T) {
	testdir, err := ioutil.TempDir("", "filestore-test")
	if err != nil {
		t.Fatal(err)
	}

	max, err := NewMaxService(&FSConfig{testdir, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
	if err != nil {
		t.Fatal(err)
	}

	max.repo.Close()

	max, err = NewMaxService(&FSConfig{testdir, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
	if err != nil {
		t.Fatal(err)
	}
}
func TestNewMaxServiceRepoLocked(t *testing.T) {
	testdir, err := ioutil.TempDir("", "filestore-test")
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewMaxService(&FSConfig{testdir, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewMaxService(&FSConfig{testdir, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
	if err == nil {
		t.Fatal(err)
	}
}

type FileConfig struct {
	path       string
	createFile bool // if true, create a file in path
	size       uint64
	prefix     string
	encrypt    bool
	password   string
	data       []byte // data used to be saved in the file when creation is needed
}

type Result struct {
	max      *MaxService
	repoRoot string
	root     ipld.Node
	list     []*helpers.UnixfsNode
	buf      []byte // file content with prefix
}

// init fs, add file, then check file content
func addFileAndCheckFileContent(initCfg *Config, fileCfg *FileConfig) (*Result, error) {
	var fname string
	var err error
	var buf []byte

	max, repoRoot, err := initFsFromConfig(initCfg)
	if err != nil {
		return nil, err
	}

	if fileCfg.createFile {
		if fileCfg.data == nil {
			buf = make([]byte, fileCfg.size)
			rand.Read(buf)
		} else {
			if uint64(len(fileCfg.data)) != fileCfg.size {
				return nil, errors.New("error in fileconfig, data size not equal to filesize")
			}

			buf = fileCfg.data
		}

		// when path is empty, create a temp file, otherwise create in the path
		if fileCfg.path == "" {
			testdir, err := ioutil.TempDir("", "filestore-test")
			if err != nil {
				return nil, err
			}
			fname, err = makeTempFile(testdir, buf)
			if err != nil {
				return nil, err
			}
		} else {
			err = makeFile(fileCfg.path, buf)
			if err != nil {
				return nil, err
			}
			fname = fileCfg.path
		}
	} else {
		//read from file
		buf, err = ioutil.ReadFile(fileCfg.path)
		if err != nil {
			return nil, err
		}

		fname = fileCfg.path
	}

	var root ipld.Node
	var list []*helpers.UnixfsNode

	// use differnt method to add
	if initCfg.fsType == FS_FILESTORE {
		root, list, err = max.NodesFromFile(fname, fileCfg.prefix, fileCfg.encrypt, fileCfg.password)
	} else {
		root, list, err = max.AddFileToFS(fname, fileCfg.prefix, fileCfg.encrypt, fileCfg.password)
	}

	if err != nil {
		return nil, err
	}

	//TODO: check list content same as file

	var cids []*cid.Cid

	if len(list) == 0 {
		cids = append(cids, root.Cid())
	} else {
		cids, err = getCidsFromNodelist(list)
		if err != nil {
			return nil, err
		}
	}

	if fileCfg.encrypt {
		tmpFile := fname + ".tmp"
		encFile := fname + ".enc"

		err = ioutil.WriteFile(tmpFile, buf, 777)
		if err != nil {
			return nil, err
		}

		err = EncryptFile(tmpFile, fileCfg.password, encFile)
		if err != nil {
			return nil, err
		}

		buf, err = ioutil.ReadFile(encFile)
		if err != nil {
			return nil, err
		}
	}

	err = checkFileContent(max, cids, getBufWithPrefix(buf, fileCfg.prefix))
	if err != nil {
		return nil, err
	}

	result := &Result{max, repoRoot, root, list, getBufWithPrefix(buf, fileCfg.prefix)}

	return result, nil
}

func TestNodesFromFileNormal(t *testing.T) {
	initCfg := &Config{"", true, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}
	fileCfg := &FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil}

	_, err := addFileAndCheckFileContent(initCfg, fileCfg)
	if err != nil {
		t.Fatal(err)
	}
}

/*
// this case is not applicable anymore since all files can be added to file store
func TestNodesFromFileSamePrefix(t *testing.T) {
	os.Mkdir("./fs", 777)

	defer func() {
		os.RemoveAll("./fs")
		os.Remove("./fstest.txt")
	}()

	initCfg := &Config{"", true, "./fs", false, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}
	fileCfgPrefixGood := &FileConfig{"./fs/test.txt", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil}
	fileCfgPrefixBad := &FileConfig{"./fstest.txt", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil}

	_, err := addFileAndCheckFileContent(initCfg, fileCfgPrefixGood)
	if err != nil {
		t.Fatal(err)
	}

	_, err = addFileAndCheckFileContent(initCfg, fileCfgPrefixBad)
	if err == nil {
		t.Fatal("invalid file path, should return error")
	} else {
		if !strings.Contains(err.Error(), "cannot add filestore references outside ont-ipfs root") {
			t.Fatal("other error")
		}
	}
}
*/

func TestNodeFromFileNotExist(t *testing.T) {
	initCfg := &Config{"", true, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}

	max, _, err := initFsFromConfig(initCfg)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = max.NodesFromFile(getCurrentDirectory()+"/"+RandStringBytes(5), RandStringBytes(20), false, "")
	if err == nil {
		t.Fatal(err)
	}
}

func TestNodesFromFileParams(t *testing.T) {
	defaultConfig := &Config{"", true, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}
	defaultFileConfig := []*FileConfig{&FileConfig{"", true, 20 * CHUNK_SIZE, RandStringBytes(20), false, "", nil}}

	cases := map[*Config][]*FileConfig{
		//different file size
		defaultConfig: []*FileConfig{
			&FileConfig{"", true, 0.5 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 10 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1000 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
		},

		//differnet chunk size
		&Config{"", true, FS_FILESTORE, CHUNK_SIZE / 256, GC_PERIOD, nil, ""}: defaultFileConfig,
		&Config{"", true, FS_FILESTORE, CHUNK_SIZE / 16, GC_PERIOD, nil, ""}:  defaultFileConfig,
		&Config{"", true, FS_FILESTORE, CHUNK_SIZE / 2, GC_PERIOD, nil, ""}:   defaultFileConfig,
		&Config{"", true, FS_FILESTORE, 2 * CHUNK_SIZE, GC_PERIOD, nil, ""}:   defaultFileConfig,

		// test encrypt file
		defaultConfig: []*FileConfig{
			&FileConfig{"", true, 0.5 * CHUNK_SIZE, RandStringBytes(20), true, RandStringBytes(6), nil},
			&FileConfig{"", true, 1 * CHUNK_SIZE, RandStringBytes(20), true, RandStringBytes(6), nil},
			&FileConfig{"", true, 10 * CHUNK_SIZE, RandStringBytes(20), true, RandStringBytes(6), nil},
			&FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), true, RandStringBytes(6), nil},
			&FileConfig{"", true, 1000 * CHUNK_SIZE, RandStringBytes(20), true, RandStringBytes(6), nil},
		},
	}

	for initCfg, fileCfgs := range cases {
		for _, fileCfg := range fileCfgs {
			_, err := addFileAndCheckFileContent(initCfg, fileCfg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestIsFileStore(t *testing.T) {
	cfg := &Config{"", true, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}
	max, _, err := initFsFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if !max.IsFileStore() {
		t.Fatalf("IsFileStore check error")
	}

	cfg = &Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}
	max, _, err = initFsFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if max.IsFileStore() {
		t.Fatalf("IsFileStore check error")
	}
}

type TagConfig struct {
	blockHash string
	fileHash  string
	index     uint64
	tag       []byte
}

func TestPutGetTag(t *testing.T) {
	testdir, err := ioutil.TempDir("", "filestore-test")
	if err != nil {
		t.Fatal(err)
	}

	max, err := NewMaxService(&FSConfig{testdir, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
	if err != nil {
		t.Fatal(err)
	}

	blockHash := RandStringBytes(20)
	fileHash := RandStringBytes(20)
	fileHash2 := RandStringBytes(20)
	index := rand.Uint64()
	index2 := rand.Uint64()

	var tag []byte
	var tag2 []byte
	var tag3 []byte
	var tag4 []byte

	rand.Read(tag)
	rand.Read(tag2)
	rand.Read(tag3)
	rand.Read(tag4)

	cases := []*TagConfig{
		&TagConfig{blockHash, fileHash, index, tag},
		&TagConfig{blockHash, fileHash, index2, tag2},
		&TagConfig{blockHash, fileHash2, index, tag3},
		&TagConfig{blockHash, fileHash, index, tag4},
	}

	for _, config := range cases {
		err = max.PutTag(config.blockHash, config.fileHash, config.index, config.tag)
		if err != nil {
			t.Fatal(err)
		}

		err = getAndCheckTag(max, config.blockHash, config.fileHash, config.index, config.tag)
		if err != nil {
			t.Fatal(err)
		}
	}

	indexes, err := max.getTagIndexes(blockHash, fileHash)
	if err != nil {
		t.Fatal(err)
	}

	if len(indexes) != 2 {
		t.Fatalf("wrong index num %d\n", len(indexes))
	}

	// check index and index2 are the indexes
	for _, value := range indexes {
		if value != index && value != index2 {
			t.Fatal("index not match")
		}
	}
}

func getAndCheckTag(max *MaxService, blockHash string, fileHash string, index uint64, tag []byte) error {

	result, err := max.GetTag(blockHash, fileHash, index)
	if err != nil {
		return err
	}

	if !bytes.Equal(tag, result) {
		return errors.New("tag mismatch with result")
	}

	return nil
}

func TestGetTagIndex(t *testing.T) {

}

func TestAddFile(t *testing.T) {
	testdir, err := ioutil.TempDir("", "filestore-test")

	if err != nil {
		t.Fatal(err)
	}

	max, err := NewMaxService(&FSConfig{testdir, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 100*CHUNK_SIZE)
	rand.Read(buf)

	fname, err := makeTempFile(testdir, buf)
	if err != nil {
		t.Fatal(err)
	}

	prefix := RandStringBytes(20)
	_, list, err := max.AddFileToFS(fname, prefix, false, "")
	if err != nil {
		t.Fatal(err)
	}

	cids, err := getCidsFromNodelist(list)
	if err != nil {
		t.Fatal(err)
	}

	err = checkFileContent(max, cids, getBufWithPrefix(buf, prefix))
	if err != nil {
		t.Fatal(err)
	}
}

// test deleteFIle cannot delete immediately when periodic gc is set
func TestDeleteFilePeriodic(t *testing.T) {
	initCfg := &Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}
	fileCfg := &FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil}

	result, err := addFileAndCheckFileContent(initCfg, fileCfg)
	if err != nil {
		t.Fatal(err)
	}

	root := result.root
	max := result.max
	list := result.list

	err = max.DeleteFile(root.Cid().String())
	if err != nil {
		t.Fatal(err)
	}

	cids, err := getCidsFromNodelist(list)
	if err != nil {
		t.Fatal(err)
	}

	err = checkFileBlocksNoExist(max, cids)
	if err == nil {
		t.Fatal(err)
	}
}
func TestDeleteFileDirect(t *testing.T) {
	initCfg := &Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD_IMMEDIATE, nil, ""}
	fileCfg := &FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil}

	result, err := addFileAndCheckFileContent(initCfg, fileCfg)
	if err != nil {
		t.Fatal(err)
	}

	root := result.root
	max := result.max
	list := result.list

	err = max.DeleteFile(root.Cid().String())
	if err != nil {
		t.Fatal(err)
	}

	cids, err := getCidsFromNodelist(list)
	if err != nil {
		t.Fatal(err)
	}

	err = checkFileBlocksNoExist(max, cids)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteFileFileStore(t *testing.T) {
	initCfg := &Config{"", true, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD_IMMEDIATE, nil, ""}
	fileCfg := &FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil}

	result, err := addFileAndCheckFileContent(initCfg, fileCfg)
	if err != nil {
		t.Fatal(err)
	}

	root := result.root
	max := result.max

	err = max.DeleteFile(root.Cid().String())
	if err == nil {
		t.Fatal(err)
	}
}

func TestPeriodicGC(t *testing.T) {
	initCfg := &Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD_TEST, nil, "26M"}
	fileCfg := &FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil}

	result, err := addFileAndCheckFileContent(initCfg, fileCfg)
	if err != nil {
		t.Fatal(err)
	}

	max := result.max
	list := result.list
	root := result.root

	cids, err := getCidsFromNodelist(list)

	err = max.DeleteFile(root.Cid().String())
	if err != nil {
		t.Fatal(err)
	}

	err = checkFileContent(max, cids, result.buf)
	if err != nil {
		t.Fatal(err)
	}

	duration, err := time.ParseDuration(GC_PERIOD_TEST)
	if err != nil {
		t.Fatal(err)
	}

	// make sure gc is scheduled
	time.Sleep(2 * duration)

	err = checkFileBlocksNoExist(max, cids)
	if err != nil {
		t.Fatal(err)
	}
}

func TestShareBlocksDeleteFile(t *testing.T) {
	testdir, err := ioutil.TempDir("", "filestore-test")
	if err != nil {
		t.Fatal(err)
	}

	max, err := NewMaxService(&FSConfig{testdir, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 100*CHUNK_SIZE)
	rand.Read(buf)

	fname, err := makeTempFile(testdir, buf)
	if err != nil {
		t.Fatal(err)
	}

	buf2 := buf[0 : len(buf)/2]
	fname2, err := makeTempFile(testdir, buf2)

	prefix1 := RandStringBytes(20)
	prefix2 := RandStringBytes(20)

	root, list, err := max.AddFileToFS(fname, prefix1, false, "")
	if err != nil {
		t.Fatal(err)
	}

	_, list2, err := max.AddFileToFS(fname2, prefix2, false, "")
	if err != nil {
		t.Fatal(err)
	}

	cids, err := getCidsFromNodelist(list)
	if err != nil {
		t.Fatal(err)
	}

	cids2, err := getCidsFromNodelist(list2)
	if err != nil {
		t.Fatal(err)
	}

	err = checkFileContent(max, cids, getBufWithPrefix(buf, prefix1))
	if err != nil {
		t.Fatal(err)
	}

	err = checkFileContent(max, cids2, getBufWithPrefix(buf2, prefix2))
	if err != nil {
		t.Fatal(err)
	}

	err = max.DeleteFile(root.Cid().String())
	if err != nil {
		t.Fatal(err)
	}

	err = checkFileContent(max, cids2, getBufWithPrefix(buf2, prefix2))
	if err != nil {
		t.Fatal(err)
	}

	//check all cids in file1 not in file2 do not exist
	for _, node1 := range cids {
		match := false
		for _, node2 := range cids2 {
			if node2.Equals(node2) {
				match = true
				break
			}
		}

		if match == false {
			_, err = max.GetBlock(node1)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

// NOTE: in real use case, prefix is wallet addrees, so it is not consider as same file in FS
func TestSameFileWithDifferentOwnerDeleteFile(t *testing.T) {
	testdir, err := ioutil.TempDir("", "filestore-test")
	if err != nil {
		t.Fatal(err)
	}

	max, err := NewMaxService(&FSConfig{testdir, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD_IMMEDIATE, ""}, nil)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 100*CHUNK_SIZE)
	rand.Read(buf)

	fname, err := makeTempFile(testdir, buf)
	if err != nil {
		t.Fatal(err)
	}

	fname2, err := makeTempFile(testdir, buf)

	prefix1 := RandStringBytes(20)
	prefix2 := RandStringBytes(20)

	root, list, err := max.AddFileToFS(fname, prefix1, false, "")
	if err != nil {
		t.Fatal(err)
	}

	root2, list2, err := max.AddFileToFS(fname2, prefix2, false, "")
	if err != nil {
		t.Fatal(err)
	}

	if root.Cid().String() == root2.Cid().String() {
		t.Fatal("same file content differnt prefix with same cid")
	}

	cids, err := getCidsFromNodelist(list)
	if err != nil {
		t.Fatal(err)
	}

	cids2, err := getCidsFromNodelist(list2)
	if err != nil {
		t.Fatal(err)
	}

	err = checkFileContent(max, cids, getBufWithPrefix(buf, prefix1))
	if err != nil {
		t.Fatal(err)
	}

	err = checkFileContent(max, cids2, getBufWithPrefix(buf, prefix2))
	if err != nil {
		t.Fatal(err)
	}

	err = max.DeleteFile(root.Cid().String())
	if err != nil {
		t.Fatal(err)
	}

	err = checkFileContent(max, cids2, getBufWithPrefix(buf, prefix2))
	if err != nil {
		t.Fatal(err)
	}

	err = max.DeleteFile(root2.Cid().String())
	if err != nil {
		t.Fatal(err)
	}

	err = checkFileBlocksNoExist(max, cids2)
	if err != nil {
		t.Fatal(err)
	}
}
func TestGetAllCids(t *testing.T) {
	defaultConfig := &Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}

	cases := map[*Config][]*FileConfig{
		//different file size for blockstore
		defaultConfig: []*FileConfig{
			&FileConfig{"", true, 0.5 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 10 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1000 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
		},
	}

	for initCfg, fileCfgs := range cases {
		for _, fileCfg := range fileCfgs {
			result, err := addFileAndCheckFileContent(initCfg, fileCfg)
			if err != nil {
				t.Fatal(err)
			}

			max := result.max
			root := result.root
			list := result.list

			cids, err := max.GetFileAllCids(context.TODO(), root.Cid())
			ok, err := compareCids(root, list, cids, false)
			if !ok {
				t.Fatal(err)
			}
		}
	}
}

func TestGetAllCidsWithOffset(t *testing.T) {
	defaultConfig := &Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}

	cases := map[*Config][]*FileConfig{
		//different file size for blockstore
		defaultConfig: []*FileConfig{
			&FileConfig{"", true, 0.5 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 10 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1000 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
		},
	}

	for initCfg, fileCfgs := range cases {
		for _, fileCfg := range fileCfgs {
			result, err := addFileAndCheckFileContent(initCfg, fileCfg)
			if err != nil {
				t.Fatal(err)
			}

			max := result.max
			root := result.root
			list := result.list
			buf := result.buf

			// NOTE: cids will not include root cid or other intermediate cids who has no data
			cids, offsets, err := max.GetFileAllCidsWithOffset(context.TODO(), root.Cid())
			if err != nil {
				t.Fatal(err)
			}

			err = checkFileContentWithOffset(max, cids, offsets, buf)
			if err != nil {
				t.Fatal(err)
			}

			ok, err := compareCids(root, list, cids, true)
			if !ok {
				t.Fatal(err)
			}
		}
	}
}

func TestGetAllCidsFileStore(t *testing.T) {
	defaultConfig := &Config{"", true, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}

	cases := map[*Config][]*FileConfig{
		//different file size for blockstore
		defaultConfig: []*FileConfig{
			&FileConfig{"", true, 0.5 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 10 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1000 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
		},
	}

	for initCfg, fileCfgs := range cases {
		for _, fileCfg := range fileCfgs {
			result, err := addFileAndCheckFileContent(initCfg, fileCfg)
			if err != nil {
				t.Fatal(err)
			}

			root := result.root
			max := result.max

			_, err = max.GetFileAllCids(context.TODO(), root.Cid())
			if err == nil {
				t.Fatal("GetFileAllCids should not be called for filestore")
			}

			_, _, err = max.GetFileAllCidsWithOffset(context.TODO(), root.Cid())
			if err == nil {
				t.Fatal("GetFileAllCidsWithOffset should not be called for filestore")
			}
		}
	}
}

func compareCids(root ipld.Node, list []*helpers.UnixfsNode, expected []*cid.Cid, rawNodeOnly bool) (bool, error) {
	var cids []*cid.Cid
	var err error

	// the expected cids are raws nodes that has data
	if rawNodeOnly {
		cids, err = getCidsFromNodelistForRawNodes(list)
		if len(root.Links()) == 0 {
			cids = append(cids, root.Cid())
		}
	} else {
		cids, err = getCidsFromNodelist(list)
		cids = append(cids, root.Cid())
	}

	if err != nil {
		return false, err
	}

	if len(expected) != len(cids) {
		return false, fmt.Errorf("cids has differnt len: len(expected)=%d, len(cids)=%d\n", len(expected), len(cids))
	}

	for _, cid1 := range expected {
		match := false
		for _, cid2 := range cids {
			if cid1.Equals(cid2) {
				match = true
				break
			}
		}

		if !match {
			return false, fmt.Errorf("cannot find matching cid for %s\n", cid1.String())
		}
	}

	return true, nil
}

func TestWriteFileNorm(t *testing.T) {
	defaultConfig := &Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}

	cases := map[*Config][]*FileConfig{
		//different file size
		defaultConfig: []*FileConfig{
			&FileConfig{"", true, 0.5 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 10 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1000 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
		},
	}

	for initCfg, fileCfgs := range cases {
		for _, fileCfg := range fileCfgs {
			result, err := addFileAndCheckFileContent(initCfg, fileCfg)
			if err != nil {
				t.Fatal(err)
			}

			max := result.max
			root := result.root
			buf := result.buf

			path := "./data"

			err = max.WriteFileFromDAG(root.Cid(), path)
			if err != nil {
				t.Fatal(err)
			}

			data, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			if !compareByteSlice(buf, data) {
				t.Fatal("data not same")
			}

			err = os.Remove(path)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestWriteFileFileStore(t *testing.T) {
	initCfg := &Config{"", true, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}
	fileCfg := &FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil}

	result, err := addFileAndCheckFileContent(initCfg, fileCfg)
	if err != nil {
		t.Fatal(err)
	}

	path := "./data"
	err = result.max.WriteFileFromDAG(result.root.Cid(), path)
	if err == nil {
		t.Fatal(err)
	}
}
func TestWriteFileInvalidPath(t *testing.T) {
	initCfg := &Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}
	fileCfg := &FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil}

	result, err := addFileAndCheckFileContent(initCfg, fileCfg)
	if err != nil {
		t.Fatal(err)
	}

	path := ".../"
	err = result.max.WriteFileFromDAG(result.root.Cid(), path)
	if err == nil {
		t.Fatal(err)
	}
}

func compareByteSlice(a []byte, b []byte) bool {
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
func TestGetAllKeysChan(t *testing.T) {
	defaultConfig := &Config{"", true, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}

	cases := map[*Config][]*FileConfig{
		//different file size
		defaultConfig: []*FileConfig{
			&FileConfig{"", true, 0.5 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 10 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1000 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
		},
	}

	for initCfg, fileCfgs := range cases {
		for _, fileCfg := range fileCfgs {
			result, err := addFileAndCheckFileContent(initCfg, fileCfg)
			if err != nil {
				t.Fatal(err)
			}

			max := result.max
			root := result.root
			list := result.list

			cids, err := getCidsFromNodelist(list)
			if err != nil {
				t.Fatal(err)
			}

			cids = append(cids, root.Cid())

			kch, err := max.AllKeysChan(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			out := make(map[string]struct{})
			for c := range kch {
				out[c.KeyString()] = struct{}{}
			}

			if len(out) != len(cids) {
				t.Fatalf("mismatch in number of entries: len(out)= %d, len(cids)=%d\n", len(out), len(cids))
			}

			for _, c := range cids {
				if _, ok := out[c.KeyString()]; !ok {
					t.Fatal("missing cid: ", c)
				}
			}
		}
	}
}

func TestPutBlockForFileStore(t *testing.T) {
	defaultConfig := &Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}

	cases := map[*Config][]*FileConfig{
		//different file size
		defaultConfig: []*FileConfig{
			&FileConfig{"", true, 0.5 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 10 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
			&FileConfig{"", true, 1000 * CHUNK_SIZE, RandStringBytes(20), false, "", nil},
		},
	}

	for initCfg, fileCfgs := range cases {
		for _, fileCfg := range fileCfgs {

			result, err := addFileAndCheckFileContent(initCfg, fileCfg)
			if err != nil {
				t.Fatal(err)
			}

			max := result.max
			buf := result.buf
			prefix := fileCfg.prefix
			root := result.root

			// the returned value is with prefix
			buf = buf[len(prefix):]

			// build the 2nd oni fs service and try put block and try read from the filestore
			testdir2, err := ioutil.TempDir("", "filestore-test")
			if err != nil {
				t.Fatal(err)
			}

			max2, err := NewMaxService(&FSConfig{testdir2, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
			if err != nil {
				t.Fatal(err)
			}

			// create the same file in 2nd fs service without building the filestore
			fname2, err := makeTempFile(testdir2, buf)
			if err != nil {
				t.Fatal(err)
			}

			// needs to set file prefix in order to get the correct file data
			max2.SetFilePrefix(fname2, prefix)
			err = max2.PutBlockForFilestore(fname2, root, 0)
			if err != nil {
				t.Fatal(err)
			}

			cids, offsets, err := max.GetFileAllCidsWithOffset(context.TODO(), root.Cid())
			if err != nil {
				t.Fatal(err)
			}

			for i, cid := range cids {
				block, err := max.GetBlock(cid)
				if err != nil {
					t.Fatal(err)
				}
				err = max2.PutBlockForFilestore(fname2, block, offsets[i])
				if err != nil {
					t.Fatal(err)
				}
			}

			err = checkFileContent(max2, cids, getBufWithPrefix(buf, prefix))
			if err != nil {
				t.Fatal(err)
			}

			// non-leaves nodes also stored in filestore
			_, err = max.GetBlock(root.Cid())
			if err != nil {
				t.Fatal(err)
			}

			// add another file and check get block for files with different prefix works
			buf2 := make([]byte, 100*CHUNK_SIZE)
			rand.Read(buf2)
			fname3, err := makeTempFile(testdir2, buf2)
			if err != nil {
				t.Fatal(err)
			}

			prefix2 := RandStringBytes(20)
			root2, list2, err := max2.GetAllNodesFromFile(fname3, prefix2, false, "")
			if err != nil {
				t.Fatal(err)
			}

			cids2, _, err := max2.buildFileStoreForFile(fname3, prefix2, root2, list2)
			if err != nil {
				t.Fatal(err)
			}

			err = checkFileContent(max2, cids2, getBufWithPrefix(buf2, prefix2))
			if err != nil {
				t.Fatal(err)
			}

			// check read file1 is still ok
			err = checkFileContent(max2, cids, getBufWithPrefix(buf, prefix))
			if err != nil {
				t.Fatal(err)
			}

			// close the repo and test read file with a new fs
			err = max2.repo.Close()
			if err != nil {
				t.Fatal(err)
			}

			max2, err = NewMaxService(&FSConfig{testdir2, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = checkFileContent(max2, cids2, getBufWithPrefix(buf2, prefix2))
			if err != nil {
				t.Fatal(err)
			}

			// check read file1 is still ok
			err = checkFileContent(max2, cids, getBufWithPrefix(buf, prefix))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestFileEncDec(t *testing.T) {
	testdir, err := ioutil.TempDir("", "filestore-test")
	if err != nil {
		t.Fatal(err)
	}

	password := RandStringBytes(32)

	buf := make([]byte, 100*CHUNK_SIZE)
	rand.Read(buf)

	fname, err := makeTempFile(testdir, buf)
	if err != nil {
		t.Fatal(err)
	}

	encFile := fname + "_enc"
	decFile := encFile + "_dec"

	err = EncryptFile(fname, password, encFile)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := decryptFileAndCheckContent(encFile, password, decFile, buf)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatalf("check decrypted context error")
	}
}

func TestFileEncDecWithBadPassWord(t *testing.T) {
	testdir, err := ioutil.TempDir("", "filestore-test")
	if err != nil {
		t.Fatal(err)
	}

	password := RandStringBytes(32)
	wrongPassword := RandStringBytes(32)

	buf := make([]byte, 100*CHUNK_SIZE)
	rand.Read(buf)

	fname, err := makeTempFile(testdir, buf)
	if err != nil {
		t.Fatal(err)
	}

	encFile := fname + "_enc"
	badDecFile := encFile + "_bad"

	err = EncryptFile(fname, password, encFile)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := decryptFileAndCheckContent(encFile, wrongPassword, badDecFile, buf)
	if err != nil {
		t.Fatal(err)
	}

	if ok {
		t.Fatalf("check decrypted context error")
	}
}

func decryptFileAndCheckContent(file string, password string, outPath string, buf []byte) (bool, error) {
	err := DecryptFile(file, password, outPath)
	if err != nil {
		return false, err
	}

	data, err := ioutil.ReadFile(outPath)
	if err != nil {
		return false, err
	}

	if len(buf) != len(data) {
		return false, nil
	}

	for i := range buf {
		if buf[i] != data[i] {
			return false, nil
		}
	}

	return true, nil
}

func TestSaveFilePrefixForFileStore(t *testing.T) {

	initCfg := &Config{"", true, FS_BLOCKSTORE, CHUNK_SIZE, GC_PERIOD, nil, ""}
	fileCfg := &FileConfig{"", true, 100 * CHUNK_SIZE, RandStringBytes(20), false, "", nil}

	result, err := addFileAndCheckFileContent(initCfg, fileCfg)
	if err != nil {
		t.Fatal(err)
	}

	max := result.max
	root := result.root
	list := result.list

	err = max.repo.Close()
	if err != nil {
		t.Fatal(err)
	}

	// close the repo and read file again to verify the prefix has been saved
	max, err = NewMaxService(&FSConfig{result.repoRoot, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var cids []*cid.Cid
	if len(list) == 0 {
		cids = append(cids, root.Cid())
	} else {
		cids, err = getCidsFromNodelist(list)
		if err != nil {
			t.Fatal(err)
		}
	}
	err = checkFileContent(max, cids, result.buf)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSaveGetProveTasks(t *testing.T) {
	testdir, err := ioutil.TempDir("", "filestore-test")
	if err != nil {
		t.Fatal(err)
	}

	max, err := NewMaxService(&FSConfig{testdir, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
	if err != nil {
		t.Fatal(err)
	}

	data := make(map[string]*fsstore.ProveParam)

	for i := 0; i < 10; i++ {
		fileHash := RandStringBytes(20)
		luckyNum := rand.Uint64()
		bakHeight := rand.Uint64()
		bakNum := rand.Uint64()
		var brokenWalletAddr [20]byte

		rand.Read(brokenWalletAddr[:])

		err = max.saveProveTask(fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr)
		if err != nil {
			t.Fatal(err)
		}

		data[fileHash] = &fsstore.ProveParam{fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr}
	}

	tasks, err := max.getProveTasks()
	if err != nil {
		t.Fatal(err)
	}

	for _, param := range tasks {
		if !compareProveParam(data[param.FileHash], param) {
			t.Fatal("error get saved file prove parameter")
		}
	}

	//try get the provetasks after reopen repo
	max.repo.Close()

	max, err = NewMaxService(&FSConfig{testdir, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
	if err != nil {
		t.Fatal(err)
	}

	tasks, err = max.getProveTasks()
	if err != nil {
		t.Fatal(err)
	}

	for _, param := range tasks {
		if !compareProveParam(data[param.FileHash], param) {
			t.Fatal("error get saved file prove parameter")
		}
	}
}

func TestDeleteProveTask(t *testing.T) {
	testdir, err := ioutil.TempDir("", "filestore-test")
	if err != nil {
		t.Fatal(err)
	}

	max, err := NewMaxService(&FSConfig{testdir, FS_FILESTORE, CHUNK_SIZE, GC_PERIOD, ""}, nil)
	if err != nil {
		t.Fatal(err)
	}

	data := make(map[string]*fsstore.ProveParam)

	for i := 0; i < 10; i++ {
		fileHash := RandStringBytes(20)
		luckyNum := rand.Uint64()
		bakHeight := rand.Uint64()
		bakNum := rand.Uint64()
		var brokenWalletAddr [20]byte

		rand.Read(brokenWalletAddr[:])

		err = max.saveProveTask(fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr)
		if err != nil {
			t.Fatal(err)
		}

		data[fileHash] = &fsstore.ProveParam{fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr}
	}

	tasks, err := max.getProveTasks()
	if err != nil {
		t.Fatal(err)
	}

	for _, param := range tasks {
		if !compareProveParam(data[param.FileHash], param) {
			t.Fatal("error get saved file prove parameter")
		}

		err = max.deleteProveTask(param.FileHash)
		if err != nil {
			t.Fatal(err)
		}

		newtask, err := max.getProveTasks()
		if err != nil {
			t.Fatal(err)
		}

		for _, param2 := range newtask {
			if param2.FileHash == param.FileHash {
				t.Fatal("failed to delete prove task")
			}
		}
	}
}

func compareProveParam(param1 *fsstore.ProveParam, param2 *fsstore.ProveParam) bool {
	return *param1 == *param2
}
