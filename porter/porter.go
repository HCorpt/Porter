package porter

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/HCorpt/porter/log"
	"go.uber.org/zap"

	"github.com/HCorpt/porter/cron"
	"github.com/HCorpt/porter/utils"
)

const (
	ArrchiveDir = ".archive"
	DepotWkDir  = "DepotDir"
	InfoName    = ".porter"
)

type DepotInfo struct {
	Name              string    `json:"name"`
	SyncSource        string    `json:"sync_source"`
	DeportLocation    string    `json:"deport_location"`
	Owner             string    `json:"owner"`
	SyncIntervalMinu  int64     `json:"sync_interval_minu"`
	ArchiveAllotedDay int32     `json:"archive_alloted_day"`
	CreatTime         time.Time `json:"creat_time"`
}

type RoundStat struct {
	LastSyncCost  time.Duration `json:"last_sync_cost"`
	LastSyncTime  time.Time     `json:"last_sync_time"`
	LastSyncBytes int64         `json:"last_sync_bytes"`
	Error         string        `json:"error,omitempty"`
}

type SyncDepot struct {
	Depot *DepotInfo      `json:"depot"`
	Stats utils.RingStats `json:"stats"`

	mtx      sync.Mutex         `json:"-"`
	CancelFn context.CancelFunc `json:"-"`
}

func (s *SyncDepot) Sync() {
	err := fmt.Errorf("")
	start := time.Now()
	syncBytes, err := s.DoSync()
	stat := &RoundStat{
		LastSyncCost:  time.Now().Sub(start),
		LastSyncTime:  start,
		LastSyncBytes: syncBytes,
		Error:         err.Error(),
	}
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.Stats.Append(utils.Stat{
		Time: time.Now(),
		Info: stat,
	})
}

func (s *SyncDepot) Clone() *SyncDepot {
	clone := &SyncDepot{}
	utils.DeepCopy(clone, s)
	return clone
}

func (s *SyncDepot) DoSync() (int64, error) {
	if err := checkDepotSpace(s.Depot.DeportLocation); err != nil {
		return 0, err
	}
	if err := initDepot(s.Depot.DeportLocation, s.Depot); err != nil {
		return 0, err
	}
	// Archive Depot
	if err := archiveDepot(s.Depot.DeportLocation); err != nil {
		return 0, err
	}
	// Depot Sync
	return doSyncDepot(s.Depot)
}

type Porter struct {
	depots map[string]*SyncDepot
	cron   *cron.Cron
	mtx    sync.Mutex
}

func NewPorter() *Porter {
	return &Porter{
		depots: map[string]*SyncDepot{},
		cron:   cron.NewCron(),
	}
}

func checkDepotSpace(root string) error {
	if info, err := os.Stat(root); os.IsNotExist(err) || !info.IsDir() {
		return fmt.Errorf("path %s not exist or indicate a file", root)
	}
	files, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	for _, file := range files {
		if file.Name() == InfoName && !file.IsDir() {
			return nil
		}
	}
	return fmt.Errorf("sync depot dir not empty, dir path %s", root)
}

func initDepot(root string, info *DepotInfo) error {
	// create info file if not exist
	infoPath, archivePath, depotPath := filepath.Join(root, InfoName), filepath.Join(root, ArrchiveDir), filepath.Join(root, DepotWkDir)
	if info, err := os.Stat(infoPath); err != nil && os.IsNotExist(err) {
		if err := utils.WriteJsonFile(infoPath, info, 0644); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil && info.IsDir() {
		return fmt.Errorf("%s is a dir", InfoName)
	}

	// create archive dir if not exist
	if err := os.Mkdir(archivePath, 0755); err != nil && !os.IsExist(err) {
		return err
	}

	// create archive file if not exist
	if err := os.Mkdir(depotPath, 0755); err != nil && !os.IsExist(err) {
		return err
	}

	return nil
}

// ensure depotPath and archive path already exist
func archiveDepot(root string) (err error) {
	wkDir, archiveDir := filepath.Join(root, DepotWkDir), filepath.Join(root, ArrchiveDir, time.Now().Format("2006.01.02~15:04:05"))
	if err = os.Mkdir(archiveDir, 0755); err != nil && !os.IsExist(err) {
		return
	}
	files := []string{}
	if files, err = utils.RecurseListFiles(wkDir); err != nil {
		return
	}
	for _, file := range files {
		srcPath, dstPath := path.Join(root, file), path.Join(root, file)
		if err = os.MkdirAll(path.Dir(dstPath), 0755); err != nil {
			return
		}
		if err = os.Link(srcPath, dstPath); err != nil {
			return
		}
	}
	return nil
}

func doSyncDepot(info *DepotInfo) (int64, error) {
	return syncDir(info.SyncSource, path.Join(info.DeportLocation, DepotWkDir))
}

// diff and copy file
func syncDir(srcRoot, dstRoot string) (syncBytes int64, err error) {
	srcFiles, dstFiles := []string{}, []string{}
	if srcFiles, err = utils.RecurseListFiles(srcRoot); err != nil {
		return
	}
	if dstFiles, err = utils.RecurseListFiles(dstRoot); err != nil {
		return
	}
	srcFilesSet, dstFilesSet := utils.StrSliceToSet(srcFiles), utils.StrSliceToSet(dstFiles)
	for file := range srcFilesSet {
		n := int64(0)
		// src have, but depot not have, copy it
		srcFile, dstFile := path.Join(srcRoot, file), path.Join(dstRoot, file)
		needSync := false
		if !dstFilesSet[file] {
			needSync = true
		} else {
			var srcFileInfo, dstFileInfo fs.FileInfo
			if srcFileInfo, err = os.Stat(srcFile); err != nil {
				log.Logger().Info("obtain file stat info failed",
					zap.String("filepath", srcFile),
					zap.String("error", err.Error()),
				)
				continue
			}
			if dstFileInfo, err = os.Stat(dstFile); err != nil {
				log.Logger().Info("obtain file stat info failed",
					zap.String("filepath", dstFile),
					zap.String("error", err.Error()),
				)
				continue
			}
			if srcFileInfo.ModTime() != dstFileInfo.ModTime() {
				needSync = true
			}
		}
		if needSync {
			if n, err = utils.CopyFiles(dstFile, srcFile); err != nil {
				log.Logger().Info("sync file failed",
					zap.String("srcFile", srcFile),
					zap.String("dstFile", dstFile),
					zap.String("error", err.Error()),
				)
			}
			syncBytes += n
		}
	}
	// dst have, but src not have: just del it
	for file := range dstFilesSet {
		if !srcFilesSet[file] {
			if err = os.Remove(file); err != nil {
				log.Logger().Info("delete file failed",
					zap.String("File", file),
					zap.String("error", err.Error()),
				)
			}
		}
	}
	return
}

func (p *Porter) SanitizeDepot(info *DepotInfo) error {
	if info.Name == "" {
		return fmt.Errorf("Depot Info Invalid: Name failed empty")
	}
	// TODO: 添加更多的检查
	// 1、子集检查， src_dir: /A/  dst_dir:/A/B 的复制结构会导致过多垃圾复制
	if info.SyncSource == "" {
		return fmt.Errorf("Sync Source Empty")
	}
	return nil
}

func (p *Porter) AddDepot(info *DepotInfo) error {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if err := p.SanitizeDepot(info); err != nil {
		return err
	}
	if _, ok := p.depots[info.Name]; ok {
		return fmt.Errorf("task all ready in workflow")
	}
	if err := checkDepotSpace(info.DeportLocation); err != nil {
		return err
	}
	ctx, cancelFn := context.WithCancel(context.Background())

	syncDepot := &SyncDepot{
		Depot:    info,
		CancelFn: cancelFn,
	}
	p.depots[info.Name] = syncDepot

	p.cron.Submit(&cron.CronTask{
		IsLoop:       true,
		ExecTime:     time.Now(),
		IntervalDura: time.Duration(info.SyncIntervalMinu) * time.Minute,
		Run:          syncDepot.Sync,
		Ctx:          ctx,
	})
	return nil
}

func (p *Porter) DeleteDepot(name string) error {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if dps, ok := p.depots[name]; ok {
		dps.CancelFn()
		delete(p.depots, name)
	}
	return nil
}

func (p *Porter) ListAllDepot() []*DepotInfo {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	depots := []*DepotInfo{}
	for _, d := range p.depots {
		mirror := &DepotInfo{}
		utils.DeepCopy(mirror, d)
		depots = append(depots, mirror)
	}
	return depots
}
