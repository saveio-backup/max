package max

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	humanize "gx/ipfs/QmPSBJL4momYnE7DcUyk2DVhD6rH488ZmHBGLbxNdhU44K/go-humanize"
	tar "gx/ipfs/QmQine7gvHncNevKtG9QXxf3nXcwSj6aDDmMm52mHofEEp/tar-utils"
	chunker "gx/ipfs/QmWo8jYc19ppG7YoTsrr2kEtLRbARTJho5oNXFTR6B7Peq/go-ipfs-chunker"
	retry "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore/retrystore"
	dssync "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore/sync"
	mh "gx/ipfs/QmZyZDi491cCNTLfAhwcaDii2Kg4pwKRkhqQzURGDvY6ua/go-multihash"
	bstore "gx/ipfs/QmaG4DZ4JaqEfvPWt5nPPgoTzhc1tr1T3f4Nu9Jpdm8ymY/go-ipfs-blockstore"
	posinfo "gx/ipfs/Qmb3jLEFAQrqdVgWUajqEyuuDoavkSq1XQXz6tWdFWF995/go-ipfs-posinfo"
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	ipld "gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
	blocks "gx/ipfs/Qmej7nf81hi2x2tvjRBF3mcp74sQyuDH4VMYDGd1YtXjb2/go-block-format"

	gc "github.com/saveio/max/pin/gc"

	//logging "gx/ipfs/QmRb5jh8z2E8hMGN2tkvs1yHynUanqnZ3UeKwgN1i9P1F8/go-log"
	offline "gx/ipfs/QmWM5HhdG5ZQNyHQ5XhMdGmV9CvLpFynQfGpTxN2MEM7Lc/go-ipfs-exchange-offline"

	"github.com/saveio/themis/common/log"

	fstore "github.com/saveio/max/filestore"

	"github.com/saveio/max/blockservice"
	"github.com/saveio/max/core"
	"github.com/saveio/max/core/corerepo"
	"github.com/saveio/max/importer/balanced"
	"github.com/saveio/max/importer/helpers"
	"github.com/saveio/max/max/crypto"
	"github.com/saveio/max/max/dbstore"
	"github.com/saveio/max/max/fsstore"
	"github.com/saveio/max/merkledag"
	"github.com/saveio/max/merkledag/traverse"
	"github.com/saveio/max/pin"
	"github.com/saveio/max/repo"
	"github.com/saveio/max/repo/config"
	"github.com/saveio/max/repo/fsrepo"
	"github.com/saveio/max/thirdparty/verifbs"
	"github.com/saveio/max/unixfs/archive"
	sdk "github.com/saveio/themis-go-sdk"
)

//var log = logging.Logger("max")
var once sync.Once

type FSType int

const (
	FS_FILESTORE = iota
	FS_BLOCKSTORE
)

const (
	MAX_RETRY_REQUEST_TIMES = 6  // max request retry times
	MAX_REQUEST_TIMEWAIT    = 10 // request time wait in second
	PROVE_FILE_INTERVAL     = 10 // 10s proves
	MAX_PROVE_FILE_ROUTINES = 10 // maximum of concurrent check prove files
)

type MaxService struct {
	blockstore   bstore.Blockstore // blockstore could be either real blockstore or filestore
	datastore    repo.Datastore
	filestore    *fstore.Filestore
	filemanager  *fstore.FileManager
	dag          ipld.DAGService
	fsstore      *fsstore.FsStore
	pinner       pin.Pinner
	provetasks   *sync.Map
	loadingtasks bool
	repo         repo.Repo
	chain        *sdk.Chain
	config       *FSConfig
	killprove    chan struct{}
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

func NewMaxService(config *FSConfig, chain *sdk.Chain) (*MaxService, error) {
	if config.FsType != FS_BLOCKSTORE && config.FsType != FS_FILESTORE {
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

	fsstore := fsstore.NewFsStore(rds)

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

	if config.FsType == FS_FILESTORE {
		filemanager = fstore.NewFileManager(d)
		// hash security
		filestore = fstore.NewFilestore(bs, filemanager)
		blockstore = bstore.NewGCBlockstore(filestore, GCLocker)
		blockstore = &verifbs.VerifBSGC{GCBlockstore: blockstore}
	}

	offlineexch := offline.Exchange(blockstore)
	bserv := blockservice.New(blockstore, offlineexch)
	dag := merkledag.NewDAGService(bserv)
	if config.FsType != FS_FILESTORE {
		pinner = pin.NewPinner(rds, dag, dag)
	}

	service := &MaxService{
		blockstore:  blockstore,
		datastore:   d,
		filestore:   filestore,
		filemanager: filemanager,
		dag:         dag,
		fsstore:     fsstore,
		pinner:      pinner,
		provetasks:  new(sync.Map),
		repo:        repo,
		chain:       chain,
		config: &FSConfig{
			RepoRoot:  config.RepoRoot,
			FsType:    config.FsType,
			ChunkSize: config.ChunkSize,
			GcPeriod:  config.GcPeriod,
		},
		killprove: make(chan struct{}),
	}

	// start periodic GC only for blockstore, if gcPeriod is 0, gc is called immediately when deleteFile
	if config.FsType == FS_BLOCKSTORE {
		err = startPeriodicGC(context.TODO(), repo, config.GcPeriod, pinner, blockstore)
		if err != nil {
			log.Errorf("[NewMaxService] startPeriodicGC error", err)
			return nil, err
		}
		err = service.loadPDPTasksOnStartup()
		if err != nil {
			log.Errorf("[NewMaxService] loadPDPTasksOnStartup error: %s", err)
			return nil, err
		}
	} else {
		err = service.loadFilePrefixesOnStartup()
		if err != nil {
			log.Errorf("[NewMaxService] loadFilePrefixesOnStartup error: %s", err)
			return nil, err
		}
	}

	return service, nil
}

func isTooManyFDError(err error) bool {
	perr, ok := err.(*os.PathError)
	if ok && perr.Err == syscall.EMFILE {
		return true
	}

	return false
}

func (this *MaxService) NodesFromFile(fileName string, filePrefix string, encrypt bool, password string) (ipld.Node, []*helpers.UnixfsNode, error) {
	fileName, err := filepath.Abs(fileName)
	if err != nil {
		log.Errorf("[NodesFromFile] get abs path error for %s, err: %s", fileName, err)
		return nil, nil, err
	}

	root, list, err := this.GetAllNodesFromFile(fileName, filePrefix, encrypt, password)
	if err != nil {
		log.Errorf("[NodesFromFile] GetAllNodesFromFile error : %s", err)
		return root, list, err
	}

	if this.IsFileStore() {
		if !encrypt {
			_, _, err = this.buildFileStoreForFile(fileName, filePrefix, root, list)
			if err != nil {
				log.Errorf("[NodesFromFile] buildFileStoreForFile error : %s", err)
				return nil, nil, err
			}
		} else {
			// when encryption is used, cannot only use filestore since we need somewhere to store the
			// encrypted file, the file is not pinned becasue it will be useless when upload file finish
			err = this.blockstore.Put(root)
			if err != nil {
				log.Errorf("[NodesFromFile] put root to block store error : %s", err)
				return nil, nil, err
			}

			for _, node := range list {
				dagNode, err := node.GetDagNode()
				if err != nil {
					log.Errorf("[NodesFromFile] GetDagNode error : %s", err)
					return nil, nil, err
				}

				err = this.blockstore.Put(dagNode)
				if err != nil {
					log.Errorf("[NodesFromFile] put dagNode to block store error : %s", err)
					return nil, nil, err
				}
			}
		}
	}

	keys := make(map[string]struct{}, 0)
	keys[root.Cid().String()] = struct{}{}

	validList := make([]*helpers.UnixfsNode, 0)
	for _, item := range list {
		lNode, err := item.GetDagNode()
		if err != nil {
			log.Errorf("[NodesFromFile] GetDagNode error : %s", err)
			return nil, nil, errors.New("item getdagnode failed")
		}
		key := lNode.Cid().String()
		if _, ok := keys[key]; !ok {
			validList = append(validList, item)
		}
	}
	return root, validList, nil
}

func (this *MaxService) GetAllNodesFromFile(fileName string, filePrefix string, encrypt bool, password string) (ipld.Node, []*helpers.UnixfsNode, error) {
	cidVer := 0
	hashFunStr := "sha2-256"
	file, err := os.Open(fileName)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	var reader io.Reader = file
	if encrypt {
		encryptedR, err := crypto.AESEncryptFileReader(file, password)
		if err != nil {
			log.Errorf("[GetAllNodesFromFile]: AESEncryptFileReader error : %s", err)
			return nil, nil, err
		}
		reader = encryptedR
	}
	// Insert prefix to identify a file
	stringReader := strings.NewReader(filePrefix)
	reader = io.MultiReader(stringReader, reader)

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

	return root, list, nil
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

	return cids, offsets, nil
}

func (this *MaxService) PutBlock(block blocks.Block) error {
	return this.blockstore.Put(block)
}

func (this *MaxService) GetBlock(cid *cid.Cid) (blocks.Block, error) {
	return this.blockstore.Get(cid)
}

func (this *MaxService) setFilePrefix(fileName string, filePrefix string) error {
	if !this.IsFileStore() {
		log.Errorf("[setFilePrefix] not a filestore")
		return errors.New("PutBlockForFilestore can be only called on filestore")
	}

	this.filemanager.SetPrefix(fileName, filePrefix)
	return nil
}

func (this *MaxService) SetFilePrefix(fileName string, filePrefix string) error {
	err := this.setFilePrefix(fileName, filePrefix)
	if err != nil {
		log.Errorf("[SetFilePrefix] setFilePrefix error: %s", err)
		return err
	}

	return this.saveFilePrefix(fileName, filePrefix)
}

func (this *MaxService) saveFilePrefix(fileName string, filePrefix string) error {
	if !this.IsFileStore() {
		log.Errorf("[saveFilePrefix] not a filestore")
		return errors.New("saveFilePrefix can be only called on filestore")
	}

	prefix := fsstore.NewFilePrefix(fileName, filePrefix)

	return this.fsstore.PutFilePrefix(fileName, prefix)
}

func (this *MaxService) getFilePrefixes() (map[string]string, error) {
	if !this.IsFileStore() {
		log.Errorf("[getFilePrefixes] not a filestore")
		return nil, errors.New("loadFilePrefixes can be only called on filestore")
	}

	prefixes, err := this.fsstore.GetFilePrefixes()
	if err != nil {
		// TO Check what will be returned if no matching data
		if err == dbstore.ErrNotFound {
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

	return nil
}

func (this *MaxService) IsFileStore() bool {
	return this.filestore != nil
}

func (this *MaxService) PutBlockForFilestore(fileName string, block blocks.Block, offset uint64) error {
	if !this.IsFileStore() {
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
	return this.blockstore.AllKeysChan(ctx)
}

func (this *MaxService) PutTag(blockHash, fileHash string, index uint64, tag []byte) error {
	attrKey := fileHash + blockHash + strconv.FormatUint(index, 10)

	attr := fsstore.NewBlockAttr(blockHash, fileHash, index, tag)
	err := this.fsstore.PutBlockAttr(attrKey, attr)
	if err != nil {
		log.Errorf("[PutTag] error putting tag : %s", err)
		return fmt.Errorf("error putting tag :%t", err)
	}

	return nil
}

func (this *MaxService) GetTag(blockHash, fileHash string, index uint64) ([]byte, error) {
	attrKey := fileHash + blockHash + strconv.FormatUint(index, 10)
	attr, err := this.fsstore.GetBlockAttr(attrKey)
	if err != nil {
		log.Errorf("[GetTag] error getting tag ：%s", err)
		return nil, err
	}

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

	return indexes, nil
}

// delete file according to fileHash, only applicable for the FS node
func (this *MaxService) DeleteFile(fileHash string) error {
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
		}
	}

	// remove the file from provetasks
	this.provetasks.Delete(fileHash)

	return this.deleteFile(fileHash)
}

func (this *MaxService) deleteFile(fileHash string) error {
	if this.IsFileStore() {
		log.Errorf("[deleteFile] not applicable for filestore")
		return errors.New("deleteFile not applicable for fileStore")
	}

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
				log.Debugf("key %s removed by GC\n", result.KeyRemoved.String())
			}
		}
	}

	return nil
}

func (this *MaxService) checkIfNeedGCNow() (bool, error) {
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

	// when gc period is 0, not periodic gc but immediate gc
	if int64(period) == 0 {
		return true, nil
	}
	return false, nil
}

// add an external file to fs, not using the filestore
func (this *MaxService) AddFileToFS(fileName, filePrefix string, encrypt bool, password string) (ipld.Node, []*helpers.UnixfsNode, error) {
	if this.IsFileStore() {
		log.Errorf("[AddFileToFS] not applicable to filestore")
		return nil, nil, errors.New("AddFileToFS not applicable to filestore")
	}

	fileName, err := filepath.Abs(fileName)
	if err != nil {
		log.Errorf("[AddFileToFS] get abs path error for %s, err : %s", fileName, err)
		return nil, nil, err
	}

	root, nodes, err := this.NodesFromFile(fileName, filePrefix, encrypt, password)
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

	return root, nodes, nil
}

// TODO: GC is expensive, so should not call immediately but periordly
func (this *MaxService) gc() <-chan gc.Result {
	return gc.GC(context.TODO(), this.blockstore.(bstore.GCBlockstore), this.datastore, this.pinner, nil)
}

func (this *MaxService) PinRoot(ctx context.Context, rootCid *cid.Cid) error {
	if this.IsFileStore() {
		log.Errorf("[PinRoot] not applicable to filestore")
		return errors.New("pinroot not applicable to filestore")
	}

	if this.pinner == nil {
		log.Errorf("[PinRoot] pinner is nil")
		return errors.New("cannot pin because pinner is nil")
	}

	this.pinner.PinWithMode(rootCid, pin.Recursive)
	return this.pinner.Flush()

}

func (this *MaxService) unpinRoot(ctx context.Context, rootCid *cid.Cid) error {
	if this.pinner == nil {
		log.Errorf("[unpinRoot] pinner is nil")
		return errors.New("cannot pin because pinner is nil")
	}

	this.pinner.Unpin(ctx, rootCid, true)
	return this.pinner.Flush()
}

func (this *MaxService) GetFileAllCids(ctx context.Context, rootCid *cid.Cid) ([]*cid.Cid, error) {
	var cids []*cid.Cid

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

	return cids, nil
}

// return the cids and corresponding file offset with the provided rootcid,
// NOTE: root cid will not be returned if it is not a leaf node with data
func (this *MaxService) GetFileAllCidsWithOffset(ctx context.Context, rootCid *cid.Cid) ([]*cid.Cid, []uint64, error) {
	var cids []*cid.Cid
	var offsets []uint64
	var offset uint64

	dagNode, err := this.checkRootForGetCid(rootCid)
	if err != nil {
		log.Errorf("[GetFileAllCidsWithOffset] checkRootForGetCid error : %s", err)
		return nil, nil, err
	}

	if dagNode == nil {
		cids = append(cids, rootCid)
		offsets = append(offsets, 0)
		return cids, offsets, nil
	}

	getCid := func(state traverse.State) error {
		node := state.Node
		if len(node.Links()) == 0 {
			cids = append(cids, state.Node.Cid())
			offsets = append(offsets, offset)
			offset += this.config.ChunkSize
		}
		return nil
	}

	err = this.traverseMerkelDag(dagNode, getCid)
	if err != nil {
		log.Errorf("[GetFileAllCidsWithOffset] traverseMerkelDag error : %s", err)
		return nil, nil, err
	}

	return cids, offsets, nil
}

func (this *MaxService) traverseMerkelDag(node ipld.Node, travFunc traverse.Func) error {
	if this.IsFileStore() {
		log.Errorf("[traverseMerkelDag] not applicable for filestore")
		return errors.New("cannot traverse offset with filestore")
	}

	options := traverse.Options{
		DAG:            this.dag,
		Order:          traverse.DFSPre,
		Func:           travFunc,
		SkipDuplicates: true,
	}

	return traverse.Traverse(node, options)
}

func (this *MaxService) checkRootForGetCid(rootCid *cid.Cid) (ipld.Node, error) {
	if this.IsFileStore() {
		log.Errorf("[checkRootForGetCid] not applicable for filestore")
		return nil, errors.New("cannot get cids with filestore")
	}

	blk, err := this.GetBlock(rootCid)
	if err != nil {
		log.Errorf("[checkRootForGetCid] GetBlock error : %s", err)
		return nil, err
	}

	dagNode, err := merkledag.DecodeProtobufBlock(blk)
	if err != nil {
		// for a small file with one node, the root is rawnode
		if _, err = merkledag.DecodeRawBlock(blk); err == nil {
			log.Errorf("[checkRootForGetCid] DecodeRawBlock ok")
			return nil, nil
		}

		log.Errorf("[checkRootForGetCid] DecodeProtobufBlock error : %s", err)
		return nil, errors.New("error decoding root for get cid")
	}

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

	return extractor.Extract(reader)
}

func (this *MaxService) Close() error {
	err := this.repo.Close()
	if err != nil {
		log.Errorf("[Close] repo close error : %s", err)
		return err
	}
	this.StopFileProve()
	return nil
}

func (this *MaxService) StopFileProve() {
	close(this.killprove)
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

func DecryptFile(file string, password string, outPath string) error {
	return crypto.AESDecryptFile(file, password, outPath)
}

func EncryptFile(file string, password string, outPath string) error {
	return crypto.AESEncryptFile(file, password, outPath)
}
