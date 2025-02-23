package sector

import (
	ldb "github.com/saveio/max/max/leveldbstore"
	"strconv"
	"testing"
)

const SECTOR_BLOCK_SIZE = 256

const SECTOR_BLOCK_COUNT = MIN_SECTOR_SIZE / SECTOR_BLOCK_SIZE

func TestSectorPersist(t *testing.T) {
	db := InitTestDB()

	manager := InitSectorManager(db)

	sectorList := []*DBSectorInfo{
		&DBSectorInfo{
			SectorId:   1,
			ProveLevel: 1,
			Size:       MIN_SECTOR_SIZE,
		},
		&DBSectorInfo{
			SectorId:   2,
			ProveLevel: 1,
			Size:       MIN_SECTOR_SIZE,
		},
		&DBSectorInfo{
			SectorId:   3,
			ProveLevel: 2,
			Size:       MIN_SECTOR_SIZE,
		},
	}

	for index, info := range sectorList {
		sector, err := manager.CreateSector(info.SectorId, info.ProveLevel, info.Size, isPlots)
		if err != nil {
			t.Fatal(err)
		}
		sectorIdStr := strconv.Itoa(int(sector.GetSectorID()))
		sectorIdStr = sectorIdStr + "-"

		fileList := []*SectorFileInfo{
			&SectorFileInfo{
				FileHash:   sectorIdStr + "file1",
				BlockCount: SECTOR_BLOCK_COUNT / 2,
				BlockSize:  SECTOR_BLOCK_SIZE,
			},
			&SectorFileInfo{
				FileHash:   sectorIdStr + "file2",
				BlockCount: SECTOR_BLOCK_COUNT / 4,
				BlockSize:  SECTOR_BLOCK_SIZE,
			},
			&SectorFileInfo{
				FileHash:   sectorIdStr + "file3",
				BlockCount: SECTOR_BLOCK_COUNT / 4,
				BlockSize:  SECTOR_BLOCK_SIZE,
			},
		}

		for _, file := range fileList {
			_, err := manager.AddFile(info.ProveLevel, file.FileHash, file.BlockCount, file.BlockSize, isPlots)
			if err != nil {
				t.Fatal(err)
			}

			// can not add a file that is already in sector as a candidate file
			_, err = manager.AddCandidateFile(info.ProveLevel, file.FileHash, file.BlockCount, file.BlockSize, isPlots)
			if err == nil {
				t.Fatalf("cannot add a file that is already in sector as a candidate")
			}
		}

		CandidateFileList := []*SectorFileInfo{
			&SectorFileInfo{
				FileHash:   sectorIdStr + "file4",
				BlockCount: SECTOR_BLOCK_COUNT / 4,
				BlockSize:  SECTOR_BLOCK_SIZE,
			},
			&SectorFileInfo{
				FileHash:   sectorIdStr + "file5",
				BlockCount: SECTOR_BLOCK_COUNT / 4,
				BlockSize:  SECTOR_BLOCK_SIZE,
			},
		}

		// try add as file as candidate when sector is full
		for _, file := range CandidateFileList {
			_, err = manager.AddCandidateFile(info.ProveLevel, file.FileHash, file.BlockCount, file.BlockSize, isPlots)
			if err == nil {
				t.Fatalf("cannot add a file as candidate, sector already full")
			}
		}

		// delete a file to make space for candidate file
		err = manager.DeleteFile(fileList[0].FileHash)
		if err != nil {
			t.Fatal(err)
		}

		// add file as candidate should be fine now
		for _, file := range CandidateFileList {
			_, err = manager.AddCandidateFile(info.ProveLevel, file.FileHash, file.BlockCount, file.BlockSize, isPlots)
			if err != nil {
				t.Fatal(err)
			}
		}

		// move a file from candidate list to sector
		err = manager.MoveCandidateFileToSector(CandidateFileList[0].FileHash)
		if err != nil {
			t.Fatal(err)
		}

		err = sector.SetFirstProveHeight(uint64(100 * index))
		if err != nil {
			t.Fatal(err)
		}
		err = sector.SetLastProveHeight(uint64(200 * index))
		if err != nil {
			t.Fatal(err)
		}
		err = sector.SetNextProveHeight(uint64(300 * index))
		if err != nil {
			t.Fatal(err)
		}
	}

	manager2 := InitSectorManager(db)
	err := manager2.LoadSectorsOnStartup()
	if err != nil {
		t.Fatal(err)
	}

	printSectorManager(manager, t)
	printSectorManager(manager2, t)
}

func TestSectorDelete(t *testing.T) {
	db := InitTestDB()

	manager := InitSectorManager(db)

	info := &DBSectorInfo{
		SectorId:   1,
		ProveLevel: 1,
		Size:       MIN_SECTOR_SIZE,
	}

	sector, err := manager.CreateSector(info.SectorId, info.ProveLevel, info.Size, isPlots)
	if err != nil {
		t.Fatal(err)
	}
	sectorIdStr := strconv.Itoa(int(sector.GetSectorID()))
	sectorIdStr = sectorIdStr + "-"

	fileList := []*SectorFileInfo{
		&SectorFileInfo{
			FileHash:   sectorIdStr + "file1",
			BlockCount: SECTOR_BLOCK_COUNT / 2,
			BlockSize:  SECTOR_BLOCK_SIZE,
		},
		&SectorFileInfo{
			FileHash:   sectorIdStr + "file2",
			BlockCount: SECTOR_BLOCK_COUNT / 4,
			BlockSize:  SECTOR_BLOCK_SIZE,
		},
		&SectorFileInfo{
			FileHash:   sectorIdStr + "file3",
			BlockCount: SECTOR_BLOCK_COUNT / 4,
			BlockSize:  SECTOR_BLOCK_SIZE,
		},
	}

	for _, file := range fileList {
		_, err := manager.AddFile(info.ProveLevel, file.FileHash, file.BlockCount, file.BlockSize, isPlots)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = manager.DeleteSector(info.SectorId)
	if err != nil {
		t.Fatal(err)
	}

	list, err := manager.loadSectorFileList(info.SectorId)
	if err != nil {
		t.Fatal(err)
	}

	if list != nil {
		t.Fatalf("fileList for deleted sector is not nil, %v", list)
	}

	param, err := manager.loadSectorProveParam(info.SectorId)
	if err != nil {
		t.Fatal(err)
	}

	if param != nil {
		t.Fatalf("fileParam for deleted sector is not nil, %v", list)
	}
}

func printSectorManager(m *SectorManager, t *testing.T) {
	t.Logf("pirntSectorManager data :\n")
	t.Logf("manager : %+v\n", m)

	m.sectorIdMap.Range(func(key, value interface{}) bool {
		id := key.(uint64)
		sector := value.(*Sector)
		t.Logf("sector %d:\n", id)
		t.Logf("sector data %+v\n", sector)

		for _, file := range sector.GetFileList() {
			t.Logf("file %s, block count %d, block size %d\n", file.FileHash, file.BlockCount, file.BlockSize)
		}

		for _, file := range sector.GetCandidateFileList() {
			t.Logf("candidate file %s, block count %d, block size %d\n", file.FileHash, file.BlockCount, file.BlockSize)
		}
		return true
	})
}

type testDB struct {
	dataMap map[string][]byte
}

func InitTestDB() *testDB {
	return &testDB{dataMap: make(map[string][]byte)}
}

func (this *testDB) PutData(key string, data []byte) error {
	this.dataMap[key] = data
	return nil
}
func (this *testDB) GetData(key string) ([]byte, error) {
	if data, exist := this.dataMap[key]; exist {
		return data, nil
	} else {
		return nil, ldb.ErrNotFound
	}
}
func (this *testDB) DeleteData(key string) error {
	delete(this.dataMap, key)
	return nil
}
