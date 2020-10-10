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

	for _, info := range sectorList {
		sector, err := manager.CreateSector(info.SectorId, info.ProveLevel, info.Size)
		if err != nil {
			t.Fatal(err)
		}
		sectorIdStr := strconv.Itoa(int(sector.sectorId))
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
			_, err := manager.AddFile(info.ProveLevel, file.FileHash, file.BlockCount, file.BlockSize)
			if err != nil {
				t.Fatal(err)
			}
		}

		err = sector.SetFirstProveHeight(100)
		if err != nil {
			t.Fatal(err)
		}
		err = sector.SetLastProveHeight(200)
		if err != nil {
			t.Fatal(err)
		}
		err = sector.SetNextProveHeight(300)
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

	sector, err := manager.CreateSector(info.SectorId, info.ProveLevel, info.Size)
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
		_, err := manager.AddFile(info.ProveLevel, file.FileHash, file.BlockCount, file.BlockSize)
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
	for id, sector := range m.sectorIdMap {
		t.Logf("sector %d:\n", id)
		t.Logf("sector data %+v\n", sector)

		for _, file := range sector.fileList {
			t.Logf("file %s, block count %d, block size %d\n", file.FileHash, file.BlockCount, file.BlockSize)
		}
	}
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
