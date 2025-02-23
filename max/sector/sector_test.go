package sector

import (
	"math/rand"
	"sort"
	"testing"
	"time"
)

const SECTOR_SIZE = 1 * 1024 * 1024 * 1024 // 1G
const BLOCK_SIZE = 256 * 1024
const MAX_BLOCK_NUM = SECTOR_SIZE / BLOCK_SIZE

type File struct {
	FileHash   string
	BlockCount uint64
	BlockSize  uint64
}

var isPlots = false

func TestSector(t *testing.T) {
	sector := InitSector(nil, 1, SECTOR_SIZE, isPlots)

	files := []File{
		File{"file1", 100, BLOCK_SIZE},
		File{"file2", 1000, BLOCK_SIZE},
		File{"file3", 200, BLOCK_SIZE},
		File{"file4", 300, BLOCK_SIZE},
	}

	for _, file := range files {
		err := sector.AddFileToSector(file.FileHash, file.BlockCount, file.BlockSize, isPlots)
		if err != nil {
			t.Fatal(err)
		}

		if !sector.IsFileInSector(file.FileHash) {
			t.Fatalf("file %s is not in sector", file.FileHash)
		}
	}

	// biggest valid file
	bigFile := File{
		FileHash:   "bigFile",
		BlockCount: MAX_BLOCK_NUM - getTotalBlockCount(files),
		BlockSize:  BLOCK_SIZE,
	}

	err := sector.AddFileToSector(bigFile.FileHash, bigFile.BlockCount, bigFile.BlockSize, isPlots)
	if err != nil {
		t.Fatal(err)
	}

	err = sector.DeleteFileFromSector(bigFile.FileHash)
	if err != nil {
		t.Fatal(err)
	}

	if sector.IsFileInSector(bigFile.FileHash) {
		t.Fatalf("file %s is still in sector", bigFile.FileHash)
	}

	bigFile.BlockCount++
	err = sector.AddFileToSector(bigFile.FileHash, bigFile.BlockCount, bigFile.BlockSize, isPlots)
	if err == nil {
		t.Fatalf("addFileToSector should fail")
	}

	// should be ok to add file with largest size
	bigFile.BlockCount--
	err = sector.AddFileToSector(bigFile.FileHash, bigFile.BlockCount, bigFile.BlockSize, isPlots)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSectorManager(t *testing.T) {
	db := InitTestDB()
	manager := InitSectorManager(db)

	sectorId := uint64(1)
	_, err := manager.CreateSector(sectorId, 1, MIN_SECTOR_SIZE, isPlots)
	if err != nil {
		t.Fatal(err)
	}

	fileHash := "file1"

	_, err = manager.AddFile(1, fileHash, 100, SECTOR_BLOCK_SIZE, isPlots)
	if err != nil {
		t.Fatal(err)
	}

	if !manager.IsFileAdded(fileHash) {
		t.Fatalf("file added test failed,expect true")
	}

	err = manager.DeleteFile(fileHash)
	if err != nil {
		t.Fatal(err)
	}

	if manager.IsFileAdded(fileHash) {
		t.Fatalf("file added test failed,expect false")
	}

	_, err = manager.AddFile(1, fileHash, 100, SECTOR_BLOCK_SIZE, isPlots)
	if err != nil {
		t.Fatal(err)
	}

	if !manager.IsFileAdded(fileHash) {
		t.Fatalf("file added test failed,expect true")
	}

	err = manager.DeleteSector(sectorId)
	if err != nil {
		t.Fatalf("deleteSector error %s", err)
	}

	/*
		if len(manager.fileSectorIdMap) != 0 {
			t.Fatalf("fileSectorIdMap not cleared")
		}
	*/
}

func TestGetFilePosBySectorIndex(t *testing.T) {
	sector := InitSector(nil, 1, SECTOR_SIZE, isPlots)

	files := []File{
		File{"file1", 100, BLOCK_SIZE},
		File{"file2", 1000, BLOCK_SIZE},
		File{"file3", 200, BLOCK_SIZE},
		File{"file4", 300, BLOCK_SIZE},
	}
	for _, file := range files {
		err := sector.AddFileToSector(file.FileHash, file.BlockCount, file.BlockSize, isPlots)
		if err != nil {
			t.Fatal(err)
		}

		if !sector.IsFileInSector(file.FileHash) {
			t.Fatalf("file %s is not in sector", file.FileHash)
		}
	}

	totalCount := getTotalBlockCount(files)

	indexes := make([]uint64, 0)

	num := 10

	rand.Seed(time.Now().Unix())
	for i := 0; i < num; i++ {
		indexes = append(indexes, uint64(rand.Int63n(int64(totalCount))))
	}

	sort.SliceStable(indexes, func(i, j int) bool {
		return indexes[i] < indexes[j]
	})

	t.Logf("indexes : %v\n", indexes)

	filePos, err := sector.GetFilePosBySectorIndexes(indexes)
	if err != nil {
		t.Fatal(err)
	}

	for _, pos := range filePos {
		t.Logf("filePos : %+v\n", pos)
	}

}
func TestGetFilePosBySectorIndexBoundary(t *testing.T) {
	sector := InitSector(nil, 1, SECTOR_SIZE, isPlots)

	files := []File{
		File{"file1", 16, BLOCK_SIZE},
	}
	for _, file := range files {
		err := sector.AddFileToSector(file.FileHash, file.BlockCount, file.BlockSize, isPlots)
		if err != nil {
			t.Fatal(err)
		}

		if !sector.IsFileInSector(file.FileHash) {
			t.Fatalf("file %s is not in sector", file.FileHash)
		}
	}

	indexes := []uint64{1, 8, 15}

	t.Logf("indexes : %v\n", indexes)

	filePos, err := sector.GetFilePosBySectorIndexes(indexes)
	if err != nil {
		t.Fatal(err)
	}

	for _, pos := range filePos {
		t.Logf("filePos : %+v\n", pos)
	}

}

func getTotalBlockCount(files []File) uint64 {
	var count uint64
	for _, file := range files {
		count += file.BlockCount
	}
	return count
}
