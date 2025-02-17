package max

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/saveio/dsp-go-sdk/consts"
	"github.com/saveio/dsp-go-sdk/core/chain"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/saveio/dsp-go-sdk/types/suffix"

	"github.com/saveio/dsp-go-sdk/types/prefix"
	"github.com/saveio/max/max/sector"

	humanize "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPSBJL4momYnE7DcUyk2DVhD6rH488ZmHBGLbxNdhU44K/go-humanize"
	tar "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmQine7gvHncNevKtG9QXxf3nXcwSj6aDDmMm52mHofEEp/tar-utils"
	mpool "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWBug6eBS7AxRdCDVuSY5CnSit7cS2XnPFYJWqWDumhCG/go-msgio/mpool"
	chunker "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWo8jYc19ppG7YoTsrr2kEtLRbARTJho5oNXFTR6B7Peq/go-ipfs-chunker"
	retry "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore/retrystore"
	dssync "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore/sync"
	mh "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZyZDi491cCNTLfAhwcaDii2Kg4pwKRkhqQzURGDvY6ua/go-multihash"
	bstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmaG4DZ4JaqEfvPWt5nPPgoTzhc1tr1T3f4Nu9Jpdm8ymY/go-ipfs-blockstore"
	posinfo "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmb3jLEFAQrqdVgWUajqEyuuDoavkSq1XQXz6tWdFWF995/go-ipfs-posinfo"
	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	ipld "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
	blocks "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmej7nf81hi2x2tvjRBF3mcp74sQyuDH4VMYDGd1YtXjb2/go-block-format"

	gc "github.com/saveio/max/pin/gc"

	//logging "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRb5jh8z2E8hMGN2tkvs1yHynUanqnZ3UeKwgN1i9P1F8/go-log"
	offline "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWM5HhdG5ZQNyHQ5XhMdGmV9CvLpFynQfGpTxN2MEM7Lc/go-ipfs-exchange-offline"

	"github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"

	fstore "github.com/saveio/max/filestore"

	"github.com/saveio/max/blockservice"
	"github.com/saveio/max/core"
	"github.com/saveio/max/core/corerepo"
	"github.com/saveio/max/importer/balanced"
	"github.com/saveio/max/importer/helpers"
	"github.com/saveio/max/max/crypto"
	"github.com/saveio/max/max/fsstore"
	"github.com/saveio/max/max/leveldbstore"
	"github.com/saveio/max/merkledag"
	"github.com/saveio/max/merkledag/traverse"
	"github.com/saveio/max/pin"
	"github.com/saveio/max/repo"
	"github.com/saveio/max/repo/config"
	"github.com/saveio/max/repo/fsrepo"
	"github.com/saveio/max/thirdparty/verifbs"
	"github.com/saveio/max/unixfs/archive"
	fscontract "github.com/saveio/themis-go-sdk/fs"
	keypair "github.com/saveio/themis/crypto/keypair"
)

//var log = logging.Logger("max")
var once sync.Once
var Version string

type FSType int

const (
	FS_FILESTORE = iota + 1
	FS_BLOCKSTORE
)

const (
	MAX_RETRY_REQUEST_TIMES                   = 3       // max request retry times
	MAX_REQUEST_TIMEWAIT                      = 5       // request time wait in second
	POLL_TX_CONFIRMED_TIMEOUT                 = 15      // timeout to poll for tx confirmed
	PROVE_FILE_INTERVAL                       = 30 * 60 // 30 mins
	PROVE_SECTOR_INTERVAL                     = 30 * 60 // 30 mins
	MAX_PROVE_FILE_ROUTINES                   = 10      // maximum of concurrent check prove files
	DEFAULT_REMOVE_NOTIFY_CHANNEL_SIZE        = 10      // default remove notify channel size
	PDP_QUEUE_SIZE                            = 50      // pdp queue size for pdp calculation and submission
	PROVE_TASK_REMOVAL_REASON_NORMAL          = "success"
	PROVE_TASK_REMOVAL_REASON_EXPIRE          = "expire"
	PROVE_TASK_REMOVAL_REASON_DELETE          = "file deleted"
	PROVE_TASK_REMOVAL_REASON_FILE_RENEWED    = "file renewed"
	PROVE_TASK_REMOVAL_REASON_PDP_CALCULATION = "pdp calculation"
)

const (
	LARGE_FILE_THRESHOLD = 256 * 1024 * 1024
)

const (
	FSSTORE_PATH       = "./fsstore"
	DirPrefixFileName  = ".SaveioDirPrefix"
	DirSealingFileName = ".SealingFile"
	EmptyBlockCid      = "SaveQmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n"
)

type ProveTaskRemovalNotify struct {
	FileHash string
	Reason   string
}

type MaxService struct {
	blockstore               bstore.Blockstore // blockstore could be either real blockstore or filestore
	datastore                repo.Datastore
	fsType                   FSType
	filestore                *fstore.Filestore
	filemanager              *fstore.FileManager
	dag                      ipld.DAGService
	fsstore                  *fsstore.FsStore
	pinner                   pin.Pinner
	provetasks               *sync.Map
	sectorProveTasks         *sync.Map
	sectorManager            *sector.SectorManager
	repo                     repo.Repo
	chain                    *chain.Chain
	config                   *FSConfig
	rpcCache                 *Cache
	pdpQueue                 *PriorityQueue
	submitQueue              *PriorityQueue
	submitting               *sync.Map
	loadingtasks             bool
	kill                     chan struct{}
	Notify                   chan *ProveTaskRemovalNotify
	chainEventNotifyChannels *sync.Map
}

type FSConfig struct {
	RepoRoot   string
	FsType     FSType
	ChunkSize  uint64
	GcPeriod   string
	MaxStorage string
}

func initRepoConfig() (*config.Config, error) {
	ident := config.Identity{PeerID: "empty", PrivKey: "empty"}
	datastore := config.DefaultDatastoreConfig()

	conf := &config.Config{
		Datastore: datastore,
		Identity:  ident,
	}

	return conf, nil
}

func setMaxStorage(repo repo.Repo, maxStorage string) error {
	size, err := humanize.ParseBytes(maxStorage)
	if err != nil {
		log.Errorf("[setMaxStorage] ParseBytes for max storage error : %s", err)
		return err
	}

	if size <= 0 {
		log.Errorf("[setMaxStorage] wrong size: %d", size)
		return errors.New("max storage value wrong")
	}

	err = repo.SetConfigKey("Datastore.StorageMax", maxStorage)
	if err != nil {
		log.Errorf("[setMaxStorage] set max storage error: %s", err)
		return err
	}

	config, err := repo.Config()
	if err != nil {
		log.Errorf("[setMaxStorage] get config error : %s", err)
		return err
	}

	config.Datastore.StorageMax = maxStorage
	log.Debugf("[setMaxStorage] set max storage : %s", maxStorage)
	return nil
}

func NewMaxService(config *FSConfig, chain *chain.Chain) (*MaxService, error) {
	if config.FsType != FS_BLOCKSTORE &&
		config.FsType != FS_FILESTORE &&
		config.FsType != (FS_BLOCKSTORE|FS_FILESTORE) {
		log.Errorf("[NewMaxService] wrong fs type : %d", config.FsType)
		return nil, errors.New("wrong fs type")
	}

	repoConfig, err := initRepoConfig()
	if err != nil {
		log.Errorf("[NewMaxService] initRepoConfig error : %s", err)
		return nil, err
	}

	locked, err := fsrepo.LockedByOtherProcess(config.RepoRoot)
	if err != nil {
		log.Errorf("[NewMaxService] LockedByOtherProcess error : %s", err)
		return nil, err
	}

	if locked {
		log.Errorf("[NewMaxService] repo locked by other process : %s", err)
		return nil, errors.New("repo locked by other process")
	}

	if !fsrepo.IsInitialized(config.RepoRoot) {
		err := fsrepo.Init(config.RepoRoot, repoConfig)
		if err != nil {
			log.Errorf("[NewMaxService] repo init error : %s", err)
			return nil, err
		}
	}

	repo, err := fsrepo.Open(config.RepoRoot)
	if err != nil {
		log.Errorf("[NewMaxService] open repo error : %s", err)
		return nil, err
	}

	if config.MaxStorage != "" {
		err = setMaxStorage(repo, config.MaxStorage)
		if err != nil {
			log.Errorf("[NewMaxService] set max storage error : %s", err)
			return nil, err
		}
	}

	d := repo.Datastore()

	rds := &retry.Datastore{
		Batching:    d,
		Delay:       time.Millisecond * 200,
		Retries:     6,
		TempErrFunc: isTooManyFDError,
	}

	fsstore, err := fsstore.NewFsStore(path.Join(config.RepoRoot, FSSTORE_PATH))
	if err != nil {
		log.Errorf("[NewMaxService] open fsstore error : %s", err)
		return nil, err
	}

	// hash security
	bs := bstore.NewBlockstore(dssync.MutexWrap(rds))
	bs = &verifbs.VerifBS{Blockstore: bs}

	opts := bstore.DefaultCacheOpts()
	cbs, err := bstore.CachedBlockstore(context.TODO(), bs, opts)
	if err != nil {
		log.Errorf("[NewMaxService] create cache block store error : %s", err)
		return nil, err
	}

	GCLocker := bstore.NewGCLocker()
	blockstore := bstore.NewGCBlockstore(cbs, GCLocker)

	var filestore *fstore.Filestore
	var filemanager *fstore.FileManager
	var pinner pin.Pinner

	if (config.FsType & FS_FILESTORE) != 0 {
		filemanager = fstore.NewFileManager(d)
		// hash security
		filestore = fstore.NewFilestore(bs, filemanager)
		blockstore = bstore.NewGCBlockstore(filestore, GCLocker)
		blockstore = &verifbs.VerifBSGC{GCBlockstore: blockstore}
	}

	offlineexch := offline.Exchange(blockstore)
	bserv := blockservice.New(blockstore, offlineexch)
	dag := merkledag.NewDAGService(bserv)

	pinner, err = pin.LoadPinner(rds, dag, dag)
	if err != nil {
		log.Debugf("Load Pinnder error : %s, create new one", err)
		pinner = pin.NewPinner(rds, dag, dag)
	}

	service := &MaxService{
		blockstore:       blockstore,
		datastore:        d,
		fsType:           config.FsType,
		filestore:        filestore,
		filemanager:      filemanager,
		dag:              dag,
		fsstore:          fsstore,
		pinner:           pinner,
		provetasks:       new(sync.Map),
		sectorProveTasks: new(sync.Map),
		sectorManager:    sector.InitSectorManager(fsstore),
		repo:             repo,
		chain:            chain,
		config: &FSConfig{
			RepoRoot:  config.RepoRoot,
			FsType:    config.FsType,
			ChunkSize: config.ChunkSize,
			GcPeriod:  config.GcPeriod,
		},
		rpcCache:                 NewCache(),
		pdpQueue:                 NewPriorityQueue(PDP_QUEUE_SIZE),
		submitQueue:              NewPriorityQueue(PDP_QUEUE_SIZE),
		submitting:               new(sync.Map),
		kill:                     make(chan struct{}),
		Notify:                   make(chan *ProveTaskRemovalNotify, DEFAULT_REMOVE_NOTIFY_CHANNEL_SIZE),
		chainEventNotifyChannels: new(sync.Map),
	}

	if service.SupportFileStore() {
		err = service.loadFilePrefixesOnStartup()
		if err != nil {
			log.Errorf("[NewMaxService] loadFilePrefixesOnStartup error: %s", err)
			return nil, err
		}
	}

	// start periodic GC only for blockstore, if gcPeriod is 0, gc is called immediately when deleteFile
	if (config.FsType & FS_BLOCKSTORE) != 0 {
		err = startPeriodicGC(context.TODO(), repo, config.GcPeriod, pinner, blockstore)
		if err != nil {
			log.Errorf("[NewMaxService] startPeriodicGC error", err)
			return nil, err
		}
		err = service.StartEventFilter(MAX_REQUEST_TIMEWAIT)
		if err != nil {
			log.Errorf("[NewMaxService] StartEventFilter error", err)
			return nil, err
		}

		err = service.sectorManager.LoadSectorsOnStartup()
		if err != nil {
			log.Errorf("[NewMaxService] LoadSectorOnStartup error", err)
			return nil, err
		}

		go service.startPdpCalculationService()
		go service.startPdpSubmissionService()
		go service.sectorManager.StartSectorManagerService()
		go service.startSectorProveService()

		err = service.loadPDPTasksOnStartup()
		if err != nil {
			log.Errorf("[NewMaxService] loadPDPTasksOnStartup error: %s", err)
			return nil, err
		}

		go service.proveFileService()
	}

	log.Debugf("[NewMaxService] new max service success")
	return service, nil
}

func isTooManyFDError(err error) bool {
	perr, ok := err.(*os.PathError)
	if ok && perr.Err == syscall.EMFILE {
		return true
	}

	return false
}

func ReturnBuffer(buffer []byte) error {
	mpool.ByteSlicePool.Put(uint32(len(buffer)), buffer)
	return nil
}

func (this *MaxService) NodesFromFile(fileName string, filePrefix string, encrypt bool,
	password string, pubKey keypair.PublicKey) (blockHashes []string, err error) {
	absFileName, err := filepath.Abs(fileName)
	if err != nil {
		log.Errorf("[NodesFromFile] get abs path error for %s, err: %s", fileName, err)
		return nil, err
	}

	fileInfo, err := os.Stat(absFileName)
	if err != nil {
		log.Errorf("[NodesFromFile] get file size error for %s, err: %s", absFileName, err)
		return nil, err
	}
	fileSize := fileInfo.Size()
	if fileSize > LARGE_FILE_THRESHOLD {
		return this.NodesFromLargeFile(fileName, filePrefix, encrypt, password)
	}

	root, list, err := this.GetAllNodesFromFile(absFileName, filePrefix, encrypt, password, pubKey)
	if err != nil {
		log.Errorf("[NodesFromFile] GetAllNodesFromFile error : %s", err)
		return nil, err
	}

	if this.SupportFileStore() && !encrypt {
		_, _, err = this.buildFileStoreForFile(absFileName, filePrefix, root, list)
		if err != nil {
			log.Errorf("[NodesFromFile] buildFileStoreForFile error : %s", err)
			return nil, err
		}
	} else {
		// when encryption is used, cannot only use filestore since we need somewhere to store the
		// encrypted file, the file is not pinned becasue it will be useless when upload file finish
		err = this.blockstore.Put(root)
		if err != nil {
			log.Errorf("[NodesFromFile] put root to block store error : %s", err)
			return nil, err
		}

		for _, node := range list {
			dagNode, err := node.GetDagNode()
			if err != nil {
				log.Errorf("[NodesFromFile] GetDagNode error : %s", err)
				return nil, err
			}

			err = this.blockstore.Put(dagNode)
			if err != nil {
				log.Errorf("[NodesFromFile] put dagNode to block store error : %s", err)
				return nil, err
			}
		}
	}

	/*
		keys := make(map[string]struct{}, 0)
		keys[root.Cid().String()] = struct{}{}

		validList := make([]*helpers.UnixfsNode, 0)
		for _, item := range list {
			lNode, err := item.GetDagNode()
			if err != nil {
				log.Errorf("[NodesFromFile] GetDagNode error : %s", err)
				return nil, errors.New("item getdagnode failed")
			}
			key := lNode.Cid().String()
			if _, ok := keys[key]; !ok {
				validList = append(validList, item)
			}
		}
	*/

	cids, err := this.GetFileAllCids(context.TODO(), root.Cid())
	if err != nil {
		log.Errorf("[]NodesFromFile getFileAllCids error : %s", err)
		return nil, err
	}

	for _, cid := range cids {
		blockHashes = append(blockHashes, cid.String())
	}

	err = this.PinRoot(context.TODO(), root.Cid())
	if err != nil {
		log.Errorf("[NodesFromFile] pinroot  error : %s", err)
		return nil, err
	}

	log.Debugf("[NodesFromFile] success for fileName : %s, filePrefix : %s, encrypt : %v", fileName, filePrefix, encrypt)
	return blockHashes, nil
}

func (this *MaxService) NodesFromDir(path string, dirPrefix string, encrypt bool,
	password string, pubKey keypair.PublicKey) (blockHashes []string, err error) {
	dirPath, err := filepath.Abs(path)
	if err != nil {
		log.Errorf("[NodesFromDir] get abs path error for %s, err: %s", path, err)
		return nil, err
	}
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		log.Errorf("[NodesFromDir] get info error for %s, err: %s", dirPath, err)
		return nil, err
	}
	if !dirInfo.IsDir() {
		log.Errorf("[NodesFromDir] %s is not a directory", dirPath)
		return nil, errors.New("not a directory")
	}
	root := &merkledag.ProtoNode{}
	list := make([]*helpers.UnixfsNode, 0)
	err = this.AddSealingFileToDag(root, list, DirPrefixFileName, dirPrefix, dirPath, encrypt)
	if err != nil {
		log.Errorf("[NodesFromDir] AddDirPrefixToDag error : %s", err)
		return nil, err
	}
	err = this.GetAllNodesFromDir(root, list, dirPath, dirPrefix, encrypt, password, pubKey, "/")
	if err != nil {
		log.Errorf("[NodesFromDir] GetAllNodesFromDir error : %s", err)
		return nil, err
	}
	cids, err := this.GetFileAllCids(context.TODO(), root.Cid())
	if err != nil {
		log.Errorf("[NodesFromDir] getFileAllCids error : %s", err)
		return nil, err
	}
	for _, c := range cids {
		blockHashes = append(blockHashes, c.String())
	}
	err = this.PinRoot(context.TODO(), root.Cid())
	if err != nil {
		log.Errorf("[NodesFromDir] pin root error : %s", err)
		return nil, err
	}
	log.Debugf("[NodesFromDir] success for fileName : %s, filePrefix : %s, encrypt : %v", path, dirPrefix, encrypt)
	return blockHashes, nil
}

func (this *MaxService) AddSealingFileToDag(root *merkledag.ProtoNode, list []*helpers.UnixfsNode, fullName string,
	content string, dirPath string, encrypt bool) error {
	reader := strings.NewReader(content)
	chunk, err := chunker.FromString(reader, fmt.Sprintf("size-%d", this.config.ChunkSize))
	if err != nil {
		log.Errorf("[AddDirPrefixToDag] chunker.FromString error : %s", err)
		return err
	}
	dagBuilder := &helpers.DagBuilderParams{
		RawLeaves: true,
		Prefix: &cid.Prefix{
			Codec:    cid.DagProtobuf,
			MhLength: -1,
			MhType:   mh.SHA2_256,
			Version:  0,
		},
		Maxlinks: helpers.DefaultLinksPerBlock,
		NoCopy:   false,
	}
	db := dagBuilder.New(chunk)
	var subRoot ipld.Node
	var subList []*helpers.UnixfsNode
	subRoot, subList, err = balanced.LayoutAndGetNodes(db)
	if err != nil {
		log.Errorf("[AddDirPrefixToDag] balanced.LayoutAndGetNodes error : %s", err)
		return err
	}
	err = root.AddNodeLink(fullName, subRoot)
	if err != nil {
		log.Errorf("[GetAllNodesFromDir]: AddNodeLink error : %s", err)
		return err
	}
	this.saveFileBlocksForDir(dirPath, "", encrypt, subRoot, subList)
	list = append(list, subList...)
	return nil
}

func (this *MaxService) NodesFromLargeFile(fileName string, filePrefix string, encrypt bool, password string) (blockHashes []string, err error) {
	absFileName, err := filepath.Abs(fileName)
	if err != nil {
		log.Errorf("[NodesFromLargeFile] get abs path error for %s, err: %s", fileName, err)
		return nil, err
	}

	db, file, err := this.PrepareHelper(fileName, filePrefix, encrypt, password)
	if err != nil {
		log.Errorf("[NodesFromLargeFile] fail to prepare DAG builder err: %s", fileName, err)
		return nil, err
	}
	defer file.Close()

	levels := db.GetMaxlevel() + 1
	lists := make([][]string, levels)

	nroot := db.NewUnixfsNode()
	db.SetPosInfo(nroot, 0)
	db.SetOffset(0)
	for !db.Done() {
		log.Debugf("[NodesFromLargeFile]: db offset : %d", db.GetOffset())

		subRoot := db.NewUnixfsNode()
		subRoot.SetLevel(db.GetMaxlevel())
		db.SetPosInfo(subRoot, db.GetOffset())

		subList, err := balanced.FillNodeRec(db, subRoot, db.GetMaxlevel(), db.GetOffset())
		if err != nil {
			log.Errorf("[NodesFromLargeFile]: FillNodeRec error : %s", err)
			return nil, err
		}
		log.Debugf("[NodesFromLargeFile]: finish one round FillNodeRec tree size: %d, len(list) : %d", subRoot.FileSize(), len(subList))

		if err := nroot.AddChild(subRoot, db); err != nil {
			return nil, err
		}

		if this.SupportFileStore() && !encrypt {
			out, err := subRoot.GetDagNode()
			if err != nil {
				log.Errorf("[NodesFromLargeFile] fail to get Dag node for subroot : %s", err)
				return nil, err
			}

			_, _, err = this.buildFileStoreForFileOffset(absFileName, filePrefix, db.GetOffset(), out, subList)
			if err != nil {
				log.Errorf("[NodesFromLargeFile] buildFileStoreForFile error : %s", err)
				return nil, err
			}

			for _, node := range subList {
				dagNode, err := node.GetDagNode()
				if err != nil {
					log.Errorf("[NodesFromLargeFile] GetDagNode error : %s", err)
					return nil, err
				}

				lists[node.GetLevel()] = append(lists[node.GetLevel()], dagNode.Cid().String())

				// return memory to mpool!
				if len(dagNode.Links()) == 0 {
					mpool.ByteSlicePool.Put(uint32(len(dagNode.RawData())), dagNode.RawData())
				}
			}
		} else {
			for _, node := range subList {
				dagNode, err := node.GetDagNode()
				if err != nil {
					log.Errorf("[NodesFromLargeFile] GetDagNode error : %s", err)
					return nil, err
				}

				err = this.blockstore.Put(dagNode)
				if err != nil {
					log.Errorf("[NodesFromLargeFile] put dagNode to block store error : %s", err)
					return nil, err
				}

				//compute hash based on cid
				lists[node.GetLevel()] = append(lists[node.GetLevel()], dagNode.Cid().String())

				// return memory to mpool!
				if len(dagNode.Links()) == 0 {
					mpool.ByteSlicePool.Put(uint32(len(dagNode.RawData())), dagNode.RawData())
				}
			}
		}

		// prepare for next round
		db.SetOffset(db.GetOffset() + subRoot.FileSize())
	}

	// get final root
	root, err := nroot.GetDagNode()
	if err != nil {
		return nil, err
	}

	// construct block hashes from top level to bottom level
	blockHashes = append(blockHashes, root.Cid().String())
	for i := levels - 1; i >= 0; i-- {
		blockHashes = append(blockHashes, lists[i]...)
	}

	log.Debugf("[NodesFromLargeFile]: return %d blockHashes", len(blockHashes))

	if this.SupportFileStore() && !encrypt {
		_, _, err = this.buildFileStoreForFileOffset(absFileName, filePrefix, db.GetOffset(), root, []*helpers.UnixfsNode{})
		if err != nil {
			log.Errorf("[NodesFromLargeFile] buildFileStoreForFile error : %s", err)
			return nil, err
		}
	} else {
		// when encryption is used, cannot only use filestore since we need somewhere to store the
		// encrypted file, the file is not pinned becasue it will be useless when upload file finish
		err = this.blockstore.Put(root)
		if err != nil {
			log.Errorf("[NodesFromLargeFile] put root to block store error : %s", err)
			return nil, err
		}
	}

	err = this.PinRoot(context.TODO(), root.Cid())
	if err != nil {
		log.Errorf("[NodesFromLargeFile] pinroot  error : %s", err)
		return nil, err
	}

	rootCid := root.Cid()
	err = this.fsstore.PutFileBlockHash(root.Cid().String(), &fsstore.FileBlockHash{rootCid.String(), blockHashes})
	if err != nil {
		log.Errorf("[NodesFromLargeFile] PutFileBlockHash error for %s, error: %s", rootCid.String(), err)
		return nil, err
	}

	log.Debugf("[NodesFromLargeFile] success for fileName : %s, filePrefix : %s, encrypt : %v", fileName, filePrefix, encrypt)
	return blockHashes, nil
}

func (this *MaxService) NodesFromLargeFileInDir(fileName string, filePrefix string, encrypt bool, password string) (ipld.Node, []*helpers.UnixfsNode, error) {
	absFileName, err := filepath.Abs(fileName)
	if err != nil {
		log.Errorf("[NodesFromLargeFile] get abs path error for %s, err: %s", fileName, err)
		return nil, nil, err
	}

	db, file, err := this.PrepareHelper(fileName, filePrefix, encrypt, password)
	if err != nil {
		log.Errorf("[NodesFromLargeFile] fail to prepare DAG builder err: %s", fileName, err)
		return nil, nil, err
	}
	defer file.Close()

	nodeList := make([]*helpers.UnixfsNode, 0)
	blockHashes := make([]string, 0)

	levels := db.GetMaxlevel() + 1
	lists := make([][]string, levels)

	nroot := db.NewUnixfsNode()
	db.SetPosInfo(nroot, 0)
	db.SetOffset(0)
	for !db.Done() {
		log.Debugf("[NodesFromLargeFile]: db offset : %d", db.GetOffset())

		subRoot := db.NewUnixfsNode()
		subRoot.SetLevel(db.GetMaxlevel())
		db.SetPosInfo(subRoot, db.GetOffset())

		subList, err := balanced.FillNodeRec(db, subRoot, db.GetMaxlevel(), db.GetOffset())
		if err != nil {
			log.Errorf("[NodesFromLargeFile]: FillNodeRec error : %s", err)
			return nil, nil, err
		}
		nodeList = append(nodeList, subList...)
		log.Debugf("[NodesFromLargeFile]: finish one round FillNodeRec tree size: %d, len(list) : %d", subRoot.FileSize(), len(subList))

		if err := nroot.AddChild(subRoot, db); err != nil {
			return nil, nil, err
		}

		if this.SupportFileStore() && !encrypt {
			out, err := subRoot.GetDagNode()
			if err != nil {
				log.Errorf("[NodesFromLargeFile] fail to get Dag node for subroot : %s", err)
				return nil, nil, err
			}

			_, _, err = this.buildFileStoreForFileOffset(absFileName, filePrefix, db.GetOffset(), out, subList)
			if err != nil {
				log.Errorf("[NodesFromLargeFile] buildFileStoreForFile error : %s", err)
				return nil, nil, err
			}

			for _, node := range subList {
				dagNode, err := node.GetDagNode()
				if err != nil {
					log.Errorf("[NodesFromLargeFile] GetDagNode error : %s", err)
					return nil, nil, err
				}

				lists[node.GetLevel()] = append(lists[node.GetLevel()], dagNode.Cid().String())

				// return memory to mpool!
				if len(dagNode.Links()) == 0 {
					mpool.ByteSlicePool.Put(uint32(len(dagNode.RawData())), dagNode.RawData())
				}
			}
		}
		// else
		for _, node := range subList {
			dagNode, err := node.GetDagNode()
			if err != nil {
				log.Errorf("[NodesFromLargeFile] GetDagNode error : %s", err)
				return nil, nil, err
			}

			err = this.blockstore.Put(dagNode)
			if err != nil {
				log.Errorf("[NodesFromLargeFile] put dagNode to block store error : %s", err)
				return nil, nil, err
			}

			//compute hash based on cid
			lists[node.GetLevel()] = append(lists[node.GetLevel()], dagNode.Cid().String())

			// return memory to mpool!
			if len(dagNode.Links()) == 0 {
				mpool.ByteSlicePool.Put(uint32(len(dagNode.RawData())), dagNode.RawData())
			}
		} // else end

		// prepare for next round
		db.SetOffset(db.GetOffset() + subRoot.FileSize())
	}

	// get final root
	root, err := nroot.GetDagNode()
	if err != nil {
		return nil, nil, err
	}

	// construct block hashes from top level to bottom level
	blockHashes = append(blockHashes, root.Cid().String())
	for i := levels - 1; i >= 0; i-- {
		blockHashes = append(blockHashes, lists[i]...)
	}

	log.Debugf("[NodesFromLargeFile]: return %d blockHashes", len(blockHashes))

	if this.SupportFileStore() && !encrypt {
		_, _, err = this.buildFileStoreForFileOffset(absFileName, filePrefix, db.GetOffset(), root, []*helpers.UnixfsNode{})
		if err != nil {
			log.Errorf("[NodesFromLargeFile] buildFileStoreForFile error : %s", err)
			return nil, nil, err
		}
	}
	// else
	// when encryption is used, cannot only use filestore since we need somewhere to store the
	// encrypted file, the file is not pinned becasue it will be useless when upload file finish
	err = this.blockstore.Put(root)
	if err != nil {
		log.Errorf("[NodesFromLargeFile] put root to block store error : %s", err)
		return nil, nil, err
	} // else end

	err = this.PinRoot(context.TODO(), root.Cid())
	if err != nil {
		log.Errorf("[NodesFromLargeFile] pinroot  error : %s", err)
		return nil, nil, err
	}

	rootCid := root.Cid()
	err = this.fsstore.PutFileBlockHash(root.Cid().String(), &fsstore.FileBlockHash{rootCid.String(), blockHashes})
	if err != nil {
		log.Errorf("[NodesFromLargeFile] PutFileBlockHash error for %s, error: %s", rootCid.String(), err)
		return nil, nil, err
	}

	log.Debugf("[NodesFromLargeFile] success for fileName : %s, filePrefix : %s, encrypt : %v", fileName, filePrefix, encrypt)
	return root, nodeList, nil
}

func (this *MaxService) PrepareHelper(fileName string, filePrefix string, encrypt bool, password string) (*helpers.DagBuilderHelper, *os.File, error) {
	cidVer := 0
	hashFunStr := "sha2-256"
	file, err := os.Open(fileName)
	if err != nil {
		log.Errorf("[PrepareHelper]: open file %s error : %s", fileName, err)
		return nil, nil, err
	}

	var reader io.Reader = file
	if encrypt {
		encryptedR, err := crypto.AESEncryptFileReader(file, password)
		if err != nil {
			file.Close()
			log.Errorf("[PrepareHelper]: AESEncryptFileReader error : %s", err)
			return nil, nil, err
		}
		reader = encryptedR
	}
	// Insert prefix to identify a file
	stringReader := strings.NewReader(filePrefix)
	reader = io.MultiReader(stringReader, reader)

	chnk, err := chunker.FromString(reader, fmt.Sprintf("size-%d", this.config.ChunkSize))
	if err != nil {
		file.Close()
		log.Errorf("[PrepareHelper]: create chunker error : %s", err)
		return nil, nil, err
	}

	prefix, err := merkledag.PrefixForCidVersion(cidVer)
	if err != nil {
		file.Close()
		log.Errorf("[PrepareHelper]: PrefixForCidVersion error : %s", err)
		return nil, nil, err
	}

	hashFunCode, _ := mh.Names[strings.ToLower(hashFunStr)]
	if err != nil {
		file.Close()
		log.Errorf("[PrepareHelper]: get hashFunCode error : %s", err)
		return nil, nil, err
	}
	prefix.MhType = hashFunCode
	prefix.MhLength = -1

	params := &helpers.DagBuilderParams{
		RawLeaves: true,
		Prefix:    &prefix,
		Maxlinks:  32,
		Maxlevel:  2,
		NoCopy:    false,
	}
	db := params.New(chnk)

	return db, file, nil
}

func (this *MaxService) GetAllNodesFromFile(fileName string, filePrefix string, encrypt bool,
	password string, pubKey keypair.PublicKey) (ipld.Node, []*helpers.UnixfsNode, error) {
	cidVer := 0
	hashFunStr := "sha2-256"
	file, err := os.Open(fileName)
	if err != nil {
		log.Errorf("[GetAllNodesFromFile]: open file %s error : %s", fileName, err)
		return nil, nil, err
	}
	defer file.Close()
	var reader io.Reader = file
	// Insert prefix to identify a file
	stringReader := strings.NewReader(filePrefix)
	reader = io.MultiReader(stringReader, reader)
	if encrypt {
		eType := prefix.ENCRYPTTYPE_NONE
		if password != "" {
			eType = prefix.ENCRYPTTYPE_AES
		}
		if pubKey != nil {
			eType = prefix.ENCRYPTTYPE_ECIES
			// password also record in filePrefix
			pwd, err := suffix.GenerateRandomPassword()
			if err != nil {
				log.Errorf("[GetAllNodesFromFile]: generate random password error : %s", err)
				return nil, nil, err
			}
			password = string(pwd)
		}
		encryptedR, err := crypto.AESEncryptFileReader(file, password)
		if err != nil {
			log.Errorf("[GetAllNodesFromFile]: AESEncryptFileReader error : %s", err)
			return nil, nil, err
		}
		reader = io.MultiReader(stringReader, encryptedR)
		// add suffix reader
		if eType == prefix.ENCRYPTTYPE_ECIES {
			ct, err := crypto.GetCipherText(pubKey, []byte(password))
			if err != nil {
				log.Errorf("[GetAllNodesFromFile]: get cipher text error : %s", err)
				return nil, nil, err
			}
			ctStr := hex.EncodeToString(ct)
			var cipherKey [suffix.SuffixLength]byte
			copy(cipherKey[:], ctStr)
			suffixReader := strings.NewReader(string(cipherKey[:]))
			reader = io.MultiReader(stringReader, encryptedR, suffixReader)
		}
	}
	chnk, err := chunker.FromString(reader, fmt.Sprintf("size-%d", this.config.ChunkSize))
	if err != nil {
		log.Errorf("[GetAllNodesFromFile]: create chunker error : %s", err)
		return nil, nil, err
	}

	prefix, err := merkledag.PrefixForCidVersion(cidVer)
	if err != nil {
		log.Errorf("[GetAllNodesFromFile]: PrefixForCidVersion error : %s", err)
		return nil, nil, err
	}

	hashFunCode, _ := mh.Names[strings.ToLower(hashFunStr)]
	if err != nil {
		log.Errorf("[GetAllNodesFromFile]: get hashFunCode error : %s", err)
		return nil, nil, err
	}
	prefix.MhType = hashFunCode
	prefix.MhLength = -1

	params := &helpers.DagBuilderParams{
		RawLeaves: true,
		Prefix:    &prefix,
		Maxlinks:  helpers.DefaultLinksPerBlock,
		NoCopy:    false,
	}
	db := params.New(chnk)

	var root ipld.Node
	var list []*helpers.UnixfsNode

	root, list, err = balanced.LayoutAndGetNodes(db)
	if err != nil {
		log.Errorf("[GetAllNodesFromFile]: LayoutAndGetNodes error : %s", err)
		return root, list, err
	}

	log.Debugf("[GetAllNodesFromFile] success for fileName : %s, filePrefix : %s, encrypt : %v", fileName, filePrefix, encrypt)
	return root, list, nil
}

func IsDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func (this *MaxService) GetAllNodesFromDir(root *merkledag.ProtoNode, list []*helpers.UnixfsNode, dirPath string,
	dirPrefix string, encrypt bool, password string, pubKey keypair.PublicKey, path string) error {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Errorf("[GetAllNodesFromDir]: open dir %s error : %s", dirPath, err)
		return err
	}
	for _, v := range files {
		if v.IsDir() {
			subDirPath := filepath.Join(dirPath, v.Name())
			subRoot := &merkledag.ProtoNode{}
			subList := make([]*helpers.UnixfsNode, 0)
			dirPre := prefix.FilePrefix{}
			dirPre.ParseFromString(dirPrefix)
			filePrefix := prefix.FilePrefix{
				Version:    prefix.PREFIX_VERSION,
				Encrypt:    encrypt,
				EncryptPwd: password,
				Owner:      dirPre.Owner,
				FileSize:   dirPre.FileSize,
				FileName:   v.Name(),
				FileType:   prefix.FILETYPE_DIR,
			}
			_ = filePrefix.MakeSalt()
			dirPrefixStr := filePrefix.String()
			err = this.GetAllNodesFromDir(subRoot, subList, subDirPath, dirPrefixStr, encrypt, password, pubKey, path+v.Name()+"/")
			if err != nil {
				log.Errorf("[GetAllNodesFromDir]: GetAllNodesFromDir error : %s", err)
				return err
			}
			_ = root.AddNodeLink(path+v.Name()+"/", subRoot)
			list = append(list, subList...)
		} else {
			filePath := filepath.Join(dirPath, v.Name())
			file, err := os.Open(filePath)
			if err != nil {
				log.Errorf("[GetAllNodesFromDir]: open file error : %s", err)
				return err
			}
			reader := io.Reader(file)
			vName := v.Name()
			if encrypt {
				eType := prefix.GetEncryptType(password, pubKey)
				vName = prefix.GetFileNameByEncryptType(vName, eType)
				reader, err = getReaderFromEncryptFile(file, dirPrefix, v.Name(), password, pubKey)
				if err != nil {
					log.Errorf("[GetAllNodesFromDir]: getReaderFromEncryptFile error : %s", err)
					return err
				}
			}
			chunk, err := chunker.FromString(reader, fmt.Sprintf("size-%d", this.config.ChunkSize))
			if err != nil {
				log.Errorf("[GetAllNodesFromDir]: create chunker error : %s", err)
				continue
			}
			dagBuilder := &helpers.DagBuilderParams{
				RawLeaves: true,
				Prefix: &cid.Prefix{
					Codec:    cid.DagProtobuf,
					MhLength: -1,
					MhType:   mh.SHA2_256,
					Version:  0,
				},
				Maxlinks: helpers.DefaultLinksPerBlock,
				NoCopy:   false,
			}
			db := dagBuilder.New(chunk)
			var subRoot ipld.Node
			var subList []*helpers.UnixfsNode
			stat, err := os.Stat(filePath)
			if stat.Size() > LARGE_FILE_THRESHOLD {
				subRoot, subList, err = this.NodesFromLargeFileInDir(filePath, "", encrypt, password)
				if err != nil {
					log.Errorf("[GetAllNodesFromDir]: nodes from large file error : %s", err)
					continue
				}
			} else {
				subRoot, subList, err = balanced.LayoutAndGetNodes(db)
				if err != nil {
					log.Errorf("[GetAllNodesFromDir]: LayoutAndGetNodes error : %s", err)
					continue
				}
			}
			this.saveFileBlocksForDir(filePath, "", encrypt, subRoot, subList)
			_ = root.AddNodeLink(path+vName, subRoot)
			list = append(list, subList...)
		}
	}
	if root.Cid().String() == EmptyBlockCid {
		filePath := filepath.Join(dirPath, DirSealingFileName)
		err := this.AddSealingFileToDag(root, list, path+DirSealingFileName, EmptyBlockCid, filePath, encrypt)
		if err != nil {
			log.Errorf("[GetAllNodesFromDir]: AddSealingFileToDag error : %s", err)
			return err
		}
	}
	this.saveFileBlocksForDir(dirPath, dirPrefix, encrypt, root, list)
	return nil
}

func getReaderFromEncryptFile(file *os.File, dirPrefix string, fileName string, password string, pubKey keypair.PublicKey) (io.Reader, error) {
	var reader io.Reader
	eType := prefix.GetEncryptType(password, pubKey)
	if eType == prefix.ENCRYPTTYPE_ECIES {
		pwd, err := suffix.GenerateRandomPassword()
		if err != nil {
			log.Errorf("[GetAllNodesFromDir]: generate random password error : %s", err)
			return nil, err
		}
		password = string(pwd)
	}
	encryptedR, err := crypto.AESEncryptFileReader(file, password)
	if err != nil {
		log.Errorf("[GetAllNodesFromFile]: AESEncryptFileReader error : %s", err)
		return nil, err
	}
	dirPre := prefix.FilePrefix{}
	dirPre.ParseFromString(dirPrefix)
	filePrefix := prefix.FilePrefix{
		Version:     prefix.PREFIX_VERSION,
		Encrypt:     true,
		EncryptPwd:  password,
		EncryptType: uint8(eType),
		Owner:       dirPre.Owner,
		FileSize:    dirPre.FileSize,
		FileName:    fileName,
		FileType:    prefix.FILETYPE_FILE,
	}
	err = filePrefix.MakeSalt()
	if err != nil {
		log.Errorf("[GetAllNodesFromDir]: make salt error : %s", err)
		return nil, err
	}
	stringReader := strings.NewReader(filePrefix.String())
	reader = io.MultiReader(stringReader, encryptedR)
	// add suffix reader
	if eType == prefix.ENCRYPTTYPE_ECIES {
		ct, err := crypto.GetCipherText(pubKey, []byte(password))
		if err != nil {
			log.Errorf("[GetAllNodesFromDir]: get cipher text error : %s", err)
			return nil, err
		}
		ctStr := hex.EncodeToString(ct)
		var cipherKey [suffix.SuffixLength]byte
		copy(cipherKey[:], ctStr)
		suffixReader := strings.NewReader(string(cipherKey[:]))
		reader = io.MultiReader(stringReader, encryptedR, suffixReader)
	}
	return reader, nil
}

func (this *MaxService) saveFileBlocksForDir(fileName string, filePrefix string, encrypt bool, root ipld.Node, list []*helpers.UnixfsNode) {
	if this.SupportFileStore() && !encrypt {
		_, _, err := this.buildFileStoreForFile(fileName, filePrefix, root, list)
		if err != nil {
			log.Errorf("[NodesFromDir] buildFileStoreForFile error : %s", err)
			return
		}
	}
	// when encryption is used, cannot only use filestore since we need somewhere to store the
	// encrypted file, the file is not pinned becasue it will be useless when upload file finish
	err := this.blockstore.Put(root)
	if err != nil {
		log.Errorf("[NodesFromDir] put root to block store error : %s", err)
		return
	}
	for _, node := range list {
		dagNode, err := node.GetDagNode()
		if err != nil {
			log.Errorf("[NodesFromDir] GetDagNode error : %s", err)
			return
		}
		err = this.blockstore.Put(dagNode)
		if err != nil {
			log.Errorf("[NodesFromDir] put dagNode to block store error : %s", err)
			return
		}
	}
}

func (this *MaxService) saveFileBlocks(fileName string, filePrefix string, encrypt bool, root ipld.Node, list []*helpers.UnixfsNode) {
	if this.SupportFileStore() && !encrypt {
		_, _, err := this.buildFileStoreForFile(fileName, filePrefix, root, list)
		if err != nil {
			log.Errorf("[NodesFromDir] buildFileStoreForFile error : %s", err)
			return
		}
	} else {
		// when encryption is used, cannot only use filestore since we need somewhere to store the
		// encrypted file, the file is not pinned becasue it will be useless when upload file finish
		err := this.blockstore.Put(root)
		if err != nil {
			log.Errorf("[NodesFromDir] put root to block store error : %s", err)
			return
		}
		for _, node := range list {
			dagNode, err := node.GetDagNode()
			if err != nil {
				log.Errorf("[NodesFromDir] GetDagNode error : %s", err)
				return
			}
			err = this.blockstore.Put(dagNode)
			if err != nil {
				log.Errorf("[NodesFromDir] put dagNode to block store error : %s", err)
				return
			}
		}
	}
}

func (this *MaxService) buildFileStoreForFile(fileName, filePrefix string, root ipld.Node, nodes []*helpers.UnixfsNode) ([]*cid.Cid, []uint64, error) {
	var offset uint64
	var offsets []uint64
	var cids []*cid.Cid
	var n ipld.Node

	err := this.SetFilePrefix(fileName, filePrefix)
	if err != nil {
		log.Errorf("[buildFileStoreForFile] SetFilePrefix error : %s", err)
		return nil, nil, err
	}

	this.blockstore.Put(root)

	for _, node := range nodes {
		dagNode, _ := node.GetDagNode()

		// no links means leaf node with content and is saved as leaf nodes
		if len(dagNode.Links()) == 0 {
			n = &posinfo.FilestoreNode{
				PosInfo: &posinfo.PosInfo{
					FullPath: fileName,
					Offset:   offset,
				},
				Node: dagNode,
			}
			cids = append(cids, dagNode.Cid())
			offsets = append(offsets, offset)
			offset += this.config.ChunkSize
		} else {
			// dagnode are stored in the backed blockstore in order to keep link information
			n = dagNode
		}

		// it is possible that we already put the cid, but it is ok,
		// since when get the cid, the content is the same even the posinfo not same
		err := this.blockstore.Put(n)
		if err != nil {
			log.Errorf("[buildFileStoreForFile] put block to block store error : %s", err)
			return nil, nil, err
		}
	}

	log.Debugf("[buildFileStoreForFile] success for fileName : %s, filePrefix : %s", fileName, filePrefix)
	return cids, offsets, nil
}

func (this *MaxService) buildFileStoreForFileOffset(fileName, filePrefix string, offset uint64, root ipld.Node, nodes []*helpers.UnixfsNode) ([]*cid.Cid, []uint64, error) {
	var offsets []uint64
	var cids []*cid.Cid
	var n ipld.Node

	err := this.SetFilePrefix(fileName, filePrefix)
	if err != nil {
		log.Errorf("[buildFileStoreForFileOffset] SetFilePrefix error : %s", err)
		return nil, nil, err
	}

	this.blockstore.Put(root)

	for _, node := range nodes {
		dagNode, _ := node.GetDagNode()

		// no links means leaf node with content and is saved as leaf nodes
		if len(dagNode.Links()) == 0 {
			n = &posinfo.FilestoreNode{
				PosInfo: &posinfo.PosInfo{
					FullPath: fileName,
					Offset:   offset,
				},
				Node: dagNode,
			}
			cids = append(cids, dagNode.Cid())
			offsets = append(offsets, offset)
			offset += this.config.ChunkSize
		} else {
			// dagnode are stored in the backed blockstore in order to keep link information
			n = dagNode
		}

		// it is possible that we already put the cid, but it is ok,
		// since when get the cid, the content is the same even the posinfo not same
		err := this.blockstore.Put(n)
		if err != nil {
			log.Errorf("[buildFileStoreForFileOffset] put block to block store error : %s", err)
			return nil, nil, err
		}
	}

	log.Debugf("[buildFileStoreForFileOffset] success for fileName : %s, filePrefix : %s", fileName, filePrefix)
	return cids, offsets, nil
}

func (this *MaxService) PutBlock(block blocks.Block) error {
	log.Debugf("[PutBlock] put block %s", block.String())
	err := this.blockstore.Put(block)
	if err != nil {
		log.Errorf("[PutBlock] put error : %s", err)
		return err
	}
	return nil
}

func (this *MaxService) GetBlock(cid *cid.Cid) (blocks.Block, error) {
	log.Debugf("[GetBlock] get block %s", cid.String())
	block, err := this.blockstore.Get(cid)
	if err != nil {
		log.Errorf("[GetBlock] get error : %s", err)
		return nil, err
	}
	return block, nil
}

func (this *MaxService) GetBlockByFileStore(cid *cid.Cid) (blocks.Block, error) {
	get, err := this.filestore.Get(cid)
	if err != nil {
		log.Errorf("[GetBlockByFileStore] get error : %s", err)
		return nil, err
	}
	return get, nil
}

func (this *MaxService) HasBlock(cid *cid.Cid) (bool, error) {
	log.Debugf("[HasBlock] has block %s", cid.String())
	bool, err := this.blockstore.Has(cid)
	if err != nil {
		log.Warnf("[HasBlock] error : %s", err)
		return bool, err
	}
	return bool, nil
}

func (this *MaxService) setFilePrefix(fileName string, filePrefix string) error {
	if !this.SupportFileStore() {
		log.Errorf("[setFilePrefix] not a filestore")
		return errors.New("setFilePrefix can be only called on filestore")
	}

	this.filemanager.SetPrefix(fileName, filePrefix)
	log.Debugf("[setFilePrefix] fileName : %s, filePrefix : %s", fileName, filePrefix)
	return nil
}

func (this *MaxService) SetFilePrefix(fileName string, filePrefix string) error {
	err := this.setFilePrefix(fileName, filePrefix)
	if err != nil {
		log.Errorf("[SetFilePrefix] setFilePrefix error: %s", err)
		return err
	}

	err = this.saveFilePrefix(fileName, filePrefix)
	if err != nil {
		log.Errorf("[SetFilePrefix] saveFilePrefix error: %s", err)
		return err
	}

	return nil
}

func (this *MaxService) saveFilePrefix(fileName string, filePrefix string) error {
	if !this.SupportFileStore() {
		log.Errorf("[saveFilePrefix] not a filestore")
		return errors.New("saveFilePrefix can be only called on filestore")
	}

	prefix := fsstore.NewFilePrefix(fileName, filePrefix)

	err := this.fsstore.PutFilePrefix(fileName, prefix)
	if err != nil {
		log.Errorf("[saveFilePrefix] PutFilePrefix error : %s", err)
	}

	log.Debugf("[saveFilePrefix] fileName: %s, filePrefix : %s", fileName, filePrefix)
	return nil
}

func (this *MaxService) getFilePrefixes() (map[string]string, error) {
	if !this.SupportFileStore() {
		log.Errorf("[getFilePrefixes] not a filestore")
		return nil, errors.New("loadFilePrefixes can be only called on filestore")
	}

	prefixes, err := this.fsstore.GetFilePrefixes()
	if err != nil {
		// TO Check what will be returned if no matching data
		if err == leveldbstore.ErrNotFound {
			log.Debugf("[getFilePrefixes] GetFilePrefixes not found error")
			return nil, nil
		}
		log.Errorf("[getFilePrefixes] GetFilePrefixes error : %s", err)
		return nil, err
	}

	pathToPrefix := make(map[string]string)

	for _, prefix := range prefixes {
		pathToPrefix[prefix.Path] = prefix.Prefix
	}

	log.Debugf("[getFilePrefixes] pathToPrefix : %v", pathToPrefix)
	return pathToPrefix, nil
}

func (this *MaxService) loadFilePrefixesOnStartup() error {
	prefixes, err := this.getFilePrefixes()
	if err != nil {
		log.Errorf("[loadFilePrefixesOnStartup] getFilePrefixes error : %s", err)
		return err
	}

	for filepath, prefix := range prefixes {
		err = this.setFilePrefix(filepath, prefix)
		if err != nil {
			log.Errorf("[loadFilePrefixesOnStartup] setFilePrefix error : %s", err)
			return err
		}
	}

	log.Debugf("[loadFilePrefixesOnStartup] success")
	return nil
}

// only for client
func (this *MaxService) IsFileStore() bool {
	return this.filestore != nil && this.fsType == FS_FILESTORE
}

// both client and server can support filestore
func (this *MaxService) SupportFileStore() bool {
	return this.filestore != nil && (this.fsType&FS_FILESTORE) != 0
}

// fs server support file prove could support blockstore only or support both
func (this *MaxService) SupportFileProve() bool {
	return (this.fsType & FS_BLOCKSTORE) != 0
}

func (this *MaxService) PutBlockForFilestore(fileName string, block blocks.Block, offset uint64) error {
	if !this.SupportFileStore() {
		log.Errorf("[PutBlockForFilestore] not a filestore")
		return errors.New("PutBlockForFilestore can be only called on filestore")
	}

	absFileName, err := filepath.Abs(fileName)
	if err != nil {
		log.Errorf("[PutBlockForFilestore] get abs path error for %s, error : %s", fileName, err)
		return err
	}

	node, err := merkledag.DecodeProtobufBlock(block)
	if err == nil {
		if len(node.Links()) == 0 {
			log.Errorf("[PutBlockForFilestore] ipld node with no link")
			return errors.New("ipld node with no link")
		}
	} else {
		node, err = merkledag.DecodeRawBlock(block)
		if err != nil {
			log.Errorf("[PutBlockForFilestore] DecodeRawBlock error : %s", err)
			return err
		}
		node = &posinfo.FilestoreNode{
			PosInfo: &posinfo.PosInfo{
				FullPath: absFileName,
				Offset:   offset,
			},
			Node: node,
		}
	}

	err = this.blockstore.Put(node)
	if err != nil {
		log.Errorf("[PutBlockForFilestore] put to block store error : %s", err)
		return err
	}

	return nil
}

func (this *MaxService) AllKeysChan(ctx context.Context) (<-chan *cid.Cid, error) {
	ch, err := this.blockstore.AllKeysChan(ctx)
	if err != nil {
		log.Errorf("[AllKeysChan] get all keys chan error : %s", err)
		return nil, err
	}
	return ch, nil
}

func (this *MaxService) PutTag(blockHash, fileHash string, index uint64, tag []byte) error {
	if len(tag) == 0 {
		log.Errorf("[PutTag] tag cannot be empty")
		return fmt.Errorf("tag is empty")
	}
	attrKey := fileHash + blockHash + strconv.FormatUint(index, 10)

	attr := fsstore.NewBlockAttr(blockHash, fileHash, index, tag)
	err := this.fsstore.PutBlockAttr(attrKey, attr)
	if err != nil {
		log.Errorf("[PutTag] error putting tag : %s", err)
		return fmt.Errorf("error putting tag :%t", err)
	}

	log.Debugf("[PutTag] success for fileHash : %s, blockHash : %s, index : %d, tag : %v", fileHash, blockHash, index, tag)
	return nil
}

func (this *MaxService) GetTag(blockHash, fileHash string, index uint64) ([]byte, error) {
	attrKey := fileHash + blockHash + strconv.FormatUint(index, 10)
	attr, err := this.fsstore.GetBlockAttr(attrKey)
	if err != nil {
		log.Errorf("[GetTag] error getting tag ：%s", err)
		return nil, err
	}

	if len(attr.Tag) == 0 {
		log.Errorf("[GetTag] tag is empty")
		return nil, fmt.Errorf("tag is empty")
	}

	log.Debugf("[GetTag] success for fileHash : %s, blockHash : %s, index : %d, tag : %v", fileHash, blockHash, index, attr.Tag)
	return attr.Tag, nil
}

func (this *MaxService) getTagIndexes(blockHash, fileHash string) ([]uint64, error) {
	var indexes []uint64

	blockAttrs, err := this.fsstore.GetBlockAttrsWithPrefix(fileHash + blockHash)
	if err != nil {
		log.Errorf("[getTagIndexes] GetBlockAttrsWithPrefix error : %s", err)
		return nil, err
	}

	for _, attr := range blockAttrs {
		indexes = append(indexes, attr.Index)
	}

	log.Debugf("[getTagIndexes] succuess for fileHash : %s, blockHash : %s, indexes : %v", fileHash, blockHash, indexes)
	return indexes, nil
}

// delete file according to fileHash, only applicable for the FS node
func (this *MaxService) DeleteFile(fileHash string) error {
	log.Debugf("[DeleteFile] delete file %s", fileHash)
	_, err := cid.Decode(fileHash)
	if err != nil {
		log.Errorf("[DeleteFile] failed to decode fileHash %s, error : %s", fileHash, err)
		return err
	}

	if this.SupportFileProve() {
		blockAttrs, err := this.fsstore.GetBlockAttrsWithPrefix(fileHash)
		if err != nil {
			log.Errorf("[DeleteFile] GetBlockAttrsWithPrefix error : %s", err)
			return err
		} else {
			for _, attr := range blockAttrs {
				key := attr.FileHash + attr.Hash + strconv.FormatUint(attr.Index, 10)
				err = this.fsstore.DeleteBlockAttr(key)
				if err != nil {
					log.Errorf("[DeleteFile] DeleteBlockAttr error : %s", err)
					return err
				}
				log.Debugf("[DeleteFile] delete tag success for fileHash : %s, blockHash : %s, index : %d",
					attr.FileHash, attr.Hash, strconv.FormatUint(attr.Index, 10))
			}
		}

		// remove the file from provetasks and db
		err = this.deleteProveTask(fileHash, true)
		if err != nil {
			log.Errorf("[DeleteFile] delete prove task error: %s", err)
		}
		log.Debugf("[DeleteFile] delete prove task for fileHash : %s", fileHash)

		err = this.fsstore.DeleteFileBlockHash(fileHash)
		if err != nil {
			log.Errorf("[DeleteFile] delete fileblockhash error: %s", err)
		}
		log.Debugf("[DeleteFile] delete fileblockhash : %s", fileHash)
		// clear rpc cache
		this.rpcCache.deleteFileInfo(fileHash)
		this.rpcCache.deleteProveDetails(fileHash)

		sectorId := this.sectorManager.GetFileSectorId(fileHash)
		if sectorId != 0 {
			err = this.sectorManager.DeleteFile(fileHash)
			if err != nil {
				log.Errorf("[DeleteFile] delete file %s from sector error: %s", fileHash, err)
			}

			sector := this.sectorManager.GetSectorBySectorId(sectorId)
			if sector != nil && sector.GetTotalBlockCount() == 0 {
				err = this.deleteSectorProveTask(sectorId)
				if err != nil {
					log.Errorf("deleteSectorProveTask error %s", err)
				}
				log.Debugf("deleteSectorProveTask for sector %d success", sectorId)
			}
		}
	}
	return this.deleteFile(fileHash)
}

func (this *MaxService) deleteFile(fileHash string) error {
	rootCid, err := cid.Decode(fileHash)
	if err != nil {
		log.Errorf("[deleteFile] decode %s error : %s", fileHash, err)
		return err
	}

	// we can do unpin since each file has a differnt root
	this.unpinRoot(context.TODO(), rootCid)

	gcNow, err := this.checkIfNeedGCNow()
	if err != nil {
		log.Errorf("[deleteFile] checkIfNeedGCNow error : %s", err)
		return err
	}

	if gcNow {
		resultChan := this.gc()

		// wait GC finish
		for result := range resultChan {
			if result.Error != nil {
				log.Errorf("[deleteFile] gc result error : %s", result.Error)
				return result.Error
			}

			if result.KeyRemoved != nil {
				log.Debugf("[deleteFile] key %s removed by GC", result.KeyRemoved.String())
			}
		}
		log.Debugf("[deleteFile] GC finish")
	} else {
		log.Debugf("[deleteFile] dont delete file now, let periodic GC do the job")
	}

	return nil
}

func (this *MaxService) checkIfNeedGCNow() (bool, error) {
	// filestore should use imeediate gc
	if this.IsFileStore() {
		return true, nil
	}

	config, err := this.repo.Config()
	if err != nil {
		log.Errorf("[checkIfNeedGCNow] get repo config error : %s", err)
		return false, err
	}

	period, err := time.ParseDuration(config.Datastore.GCPeriod)
	if err != nil {
		log.Errorf("[checkIfNeedGCNow] ParseDuration error : %s", err)
		return false, err
	}

	log.Debugf("[checkIfNeedGCNow] GC period : %d", period)

	// when gc period is 0, not periodic gc but immediate gc
	if int64(period) == 0 {
		return true, nil
	}
	return false, nil
}

// add an external file to fs, not using the filestore
func (this *MaxService) AddFileToFS(fileName, filePrefix string, encrypt bool,
	password string, pubKey keypair.PublicKey) (ipld.Node, []*helpers.UnixfsNode, error) {
	if this.IsFileStore() {
		log.Errorf("[AddFileToFS] not applicable to filestore")
		return nil, nil, errors.New("AddFileToFS not applicable to filestore")
	}

	fileName, err := filepath.Abs(fileName)
	if err != nil {
		log.Errorf("[AddFileToFS] get abs path error for %s, err : %s", fileName, err)
		return nil, nil, err
	}

	root, nodes, err := this.GetAllNodesFromFile(fileName, filePrefix, encrypt, password, pubKey)
	if err != nil {
		log.Errorf("[AddFileToFS] NodesFromFile error : %s", err)
		return nil, nil, err
	}

	this.blockstore.Put(root)

	for _, node := range nodes {
		dagNode, _ := node.GetDagNode()

		err = this.blockstore.Put(dagNode)
		if err != nil {
			log.Errorf("[AddFileToFS] put block to block store error : %s", err)
			return nil, nil, err
		}
	}

	err = this.PinRoot(context.TODO(), root.Cid())
	if err != nil {
		log.Errorf("[AddFileToFS] pinroot  error : %s", err)
		return nil, nil, err
	}

	log.Debugf("[AddFileToFS] success for fileName : %s, filePrefix : %s, encrypt : %v", fileName, filePrefix, encrypt)
	return root, nodes, nil
}

// TODO: GC is expensive, so should not call immediately but periordly
func (this *MaxService) gc() <-chan gc.Result {
	log.Debugf("[gc] gc is called")
	return gc.GC(context.TODO(), this.blockstore.(bstore.GCBlockstore), this.datastore, this.pinner, nil)
}

func (this *MaxService) PinRoot(ctx context.Context, rootCid *cid.Cid) error {
	if this.pinner == nil {
		log.Errorf("[PinRoot] pinner is nil")
		return errors.New("cannot pin because pinner is nil")
	}

	this.pinner.PinWithMode(rootCid, pin.Recursive)
	err := this.pinner.Flush()
	if err != nil {
		log.Errorf("[PinRoot] flush error : %s", err)
	}
	return nil
}

func (this *MaxService) unpinRoot(ctx context.Context, rootCid *cid.Cid) error {
	if this.pinner == nil {
		log.Errorf("[unpinRoot] pinner is nil")
		return errors.New("cannot pin because pinner is nil")
	}

	this.pinner.Unpin(ctx, rootCid, true)
	err := this.pinner.Flush()
	if err != nil {
		log.Errorf("[unpinRoot] flush error : %s", err)
	}

	log.Debugf("[unpinRoot] unpin root for %s", rootCid.String())
	return nil
}

func (this *MaxService) GetFileAllCids(ctx context.Context, rootCid *cid.Cid) ([]*cid.Cid, error) {
	var cids []*cid.Cid
	var blockHashes []string

	hashes, err := this.fsstore.GetFileBlockHashes(rootCid.String())
	if err == nil {
		for _, blockHash := range hashes.BlockHashes {
			cid, err := cid.Decode(blockHash)
			if err != nil {
				log.Errorf("[GetFileAllCids] Decode blockhash %s error : %s", blockHash, err)
				return nil, err
			}

			cids = append(cids, cid)
		}
		return cids, nil
	} else if err != leveldbstore.ErrNotFound {
		log.Errorf("[GetFileAllCids] GetFileBlockHashes error : %s", err)
		return nil, err
	}

	dagNode, err := this.checkRootForGetCid(rootCid)
	if err != nil {
		log.Errorf("[GetFileAllCids] checkRootForGetCid error : %s", err)
		return nil, err
	}

	if dagNode == nil {
		cids = append(cids, rootCid)
		return cids, nil
	}

	getCid := func(state traverse.State) error {
		cids = append(cids, state.Node.Cid())
		return nil
	}

	err = this.traverseMerkelDag(dagNode, getCid)
	if err != nil {
		log.Errorf("[GetFileAllCids] traverseMerkelDag error : %s", err)
		return nil, err
	}

	for _, cid := range cids {
		blockHashes = append(blockHashes, cid.String())
	}

	err = this.fsstore.PutFileBlockHash(rootCid.String(), &fsstore.FileBlockHash{rootCid.String(), blockHashes})
	if err != nil {
		log.Errorf("[GetFileAllCids] PutFileBlockHash error for %s, error: %s", rootCid.String(), err)
		return nil, err
	}

	log.Debugf("[GetFileAllCids] success for cid: %s", rootCid.String())
	return cids, nil
}

// return the cids and corresponding file offset with the provided rootcid,
// NOTE: root cid will not be returned if it is not a leaf node with data
func (this *MaxService) GetFileAllCidsWithOffset(ctx context.Context, rootCid *cid.Cid) (cids []*cid.Cid, offsets []uint64, indexes []uint64, err error) {
	var offset uint64
	var index uint64

	dagNode, err := this.checkRootForGetCid(rootCid)
	if err != nil {
		log.Errorf("[GetFileAllCidsWithOffset] checkRootForGetCid error : %s", err)
		return nil, nil, nil, err
	}

	if dagNode == nil {
		cids = append(cids, rootCid)
		offsets = append(offsets, 0)
		indexes = append(indexes, 0)
		return cids, offsets, indexes, nil
	}

	getCid := func(state traverse.State) error {
		node := state.Node
		if len(node.Links()) == 0 {
			cids = append(cids, state.Node.Cid())
			offsets = append(offsets, offset)
			offset += this.config.ChunkSize
			indexes = append(indexes, index)
		}
		index++
		return nil
	}

	err = this.traverseMerkelDag(dagNode, getCid)
	if err != nil {
		log.Errorf("[GetFileAllCidsWithOffset] traverseMerkelDag error : %s", err)
		return nil, nil, nil, err
	}

	log.Debugf("[GetFileAllCidsWithOffset] success for cid: %s", rootCid.String())
	return cids, offsets, indexes, nil
}

func (this *MaxService) traverseMerkelDag(node ipld.Node, travFunc traverse.Func) error {
	// dont skip duplicates, otherwise the offset will be wrong if file has duplicates
	options := traverse.Options{
		DAG:            this.dag,
		Order:          traverse.BFS,
		Func:           travFunc,
		SkipDuplicates: false,
		LightLeafNode:  true,
		ReturnBuffer:   ReturnBuffer,
	}

	err := traverse.Traverse(node, options)
	if err != nil {
		log.Debugf("[traverseMerkelDag] traverse error : %s", err)
	}
	return nil
}

func (this *MaxService) checkRootForGetCid(rootCid *cid.Cid) (ipld.Node, error) {
	blk, err := this.GetBlock(rootCid)
	if err != nil {
		log.Errorf("[checkRootForGetCid] GetBlock error : %s", err)
		return nil, err
	}

	dagNode, err := merkledag.DecodeProtobufBlock(blk)
	if err != nil {
		// for a small file with one node, the root is rawnode
		if _, err = merkledag.DecodeRawBlock(blk); err == nil {
			log.Debugf("[checkRootForGetCid] DecodeRawBlock ok")
			return nil, nil
		}

		log.Errorf("[checkRootForGetCid] DecodeProtobufBlock error : %s", err)
		return nil, errors.New("error decoding root for get cid")
	}

	log.Debugf("[checkRootForGetCid] success for cid : %s", rootCid.String())
	return dagNode, nil
}

// write the blocks stored in merkle dag to a external file
func (this *MaxService) WriteFileFromDAG(rootCid *cid.Cid, outPath string) error {
	var dagNode ipld.Node

	if this.IsFileStore() {
		log.Errorf("[WriteFileFromDAG] not applicable for filesotre")
		return errors.New("cannot write file from dag with filestore")
	}

	blk, err := this.GetBlock(rootCid)
	if err != nil {
		log.Errorf("[WriteFileFromDAG] GetBlock error : %s", err)
		return err
	}

	dagNode, err = merkledag.DecodeProtobufBlock(blk)
	if err != nil {
		if dagNode, err = merkledag.DecodeRawBlock(blk); err != nil {
			log.Debugf("[WriteFileFromDAG] DecodeRawBlock error : %s", err)
			return err
		}
		log.Errorf("[WriteFileFromDAG] DecodeProtobufBlock error : %s", err)
	}

	reader, err := archive.DagArchive(context.TODO(), dagNode, "", this.dag, false, 0)
	extractor := tar.Extractor{Path: outPath}

	err = extractor.Extract(reader)
	if err != nil {
		log.Errorf("[WriteFileFromDAG] extract error : %s", err)
		return err
	}
	return nil
}

func (this *MaxService) Close() error {
	err := this.repo.Close()
	if err != nil {
		log.Errorf("[Close] repo close error : %s", err)
		return err
	}
	err = this.fsstore.Close()
	if err != nil {
		log.Errorf("[Close] fsstore close error : %s", err)
		return err
	}
	this.StopFileProve()
	log.Debugf("[Close] success")
	return nil
}

func (this *MaxService) StopFileProve() {
	close(this.kill)
	log.Debugf("[StopFileProve] stop prove task")
}

func (this *MaxService) SetFileBlockHashes(fileHash string, blockHashes []string) error {
	_, err := cid.Decode(fileHash)
	if err != nil {
		log.Errorf("[SetFileBlockHashes] failed to decode fileHash %s, error : %s", fileHash, err)
		return err
	}

	if len(blockHashes) == 0 || fileHash != blockHashes[0] {
		log.Errorf("[SetFileBlockHashes] invalid block hashes for fileHash %s", fileHash)
		return errors.New("SetFileBlockHashes invalid block hashes")
	}

	err = this.fsstore.PutFileBlockHash(fileHash, &fsstore.FileBlockHash{fileHash, blockHashes})
	if err != nil {
		log.Errorf("[SetFileBlockHashes] PutFileBlockHash error for %s, error: %s", fileHash, err)
		return err
	}
	return nil
}

func (this *MaxService) getAccoutAddress() common.Address {
	if this.chain.CurrentAccount() == nil {
		return common.Address{}
	}
	return this.chain.Address()
}

func (this *MaxService) getFsContract() *fscontract.Fs {
	switch this.chain.GetChainType() {
	case consts.DspModeOp:
		return this.chain.GetSDK().EVM.Fs
	default:
		return this.chain.GetSDK().Native.Fs
	}
}

func startPeriodicGC(ctx context.Context, repo repo.Repo, gcPeriod string, pinner pin.Pinner, blockstore bstore.Blockstore) error {
	if _, ok := blockstore.(bstore.GCBlockstore); !ok {
		log.Errorf("[startPeriodicGC] wrong blockstore type, cannot run GC")
		return errors.New("wrong blockstore type, cannot run GC")
	}

	if gcPeriod != "" {
		_, err := time.ParseDuration(gcPeriod)
		if err != nil {
			log.Errorf("[startPeriodicGC] ParseDuration %s error : %s", gcPeriod, err)
			return fmt.Errorf("error in parse gc period : %s", err)
		}

		err = repo.SetConfigKey("Datastore.GCPeriod", gcPeriod)
		if err != nil {
			log.Errorf("[startPeriodicGC] set gc period error : %s", err)
			return err
		}

		config, err := repo.Config()
		if err != nil {
			log.Errorf("[startPeriodicGC] get repo config error : %s", err)
			return err
		}
		config.Datastore.GCPeriod = gcPeriod
		log.Debugf("[startPeriodicGC] peirod : %d", gcPeriod)
	}

	//build the fake "node" in order to use GC interface
	node := &core.IpfsNode{
		Blockstore: blockstore.(bstore.GCBlockstore),
		Repo:       repo,
		Pinning:    pinner,
	}

	go func() {
		err := corerepo.PeriodicGC(ctx, node)
		if err != nil {
			log.Error("[startPeriodicGC] PeriodicGC error : %s", err)
		}
	}()

	return nil
}

func DecryptFile(file, prefix, password, outPath string) error {
	return crypto.AESDecryptFile(file, prefix, password, outPath)
}

func EncryptFile(file, password, outPath string) error {
	return crypto.AESEncryptFile(file, password, outPath)
}

func DecryptFileA(file, prefix, outPath string, priKey keypair.PrivateKey) error {
	return crypto.ECIESDecryptFile(file, prefix, outPath, priKey)
}

func EncryptFileA(file, outPath string, pubKey keypair.PublicKey) error {
	return crypto.ECIESEncryptFile(file, outPath, pubKey)
}
