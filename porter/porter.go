package porter

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/HCorpt/porter/cron"
)

const (
	ArrchiveDir = ".archive"
	InfoName    = ".porter"
)

type DepotInfo struct {
	Name              string    `json:"name"`
	SyncSource        string    `json:"sync_source"`
	DeportLocation    string    `json:"deport_location"`
	Owner             string    `json:"owner"`
	SyncIntervalMinu  int64     `json:"sync_interval_minu"`
	ArchiveAllotedDay int32     `json:"archive_alloted_day"`
	Error             string    `json:"error,omitempty"`
	CreatTime         time.Time `json:"creat_time"`
}

type SyncDepot struct {
	Depot        *DepotInfo    `json:"depot"`
	LastSyncCost time.Duration `json:"last_sync_cost"`
	LastSyncTime time.Time     `json:"last_sync_time"`
	CancelFn     context.CancelFunc
}

func (s *SyncDepot) Sync() {
	if err := CheckDepotSpace(s.Depot.DeportLocation); err != nil {
	}
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

func CheckDepotSpace(path string) error {
	if info, err := os.Stat(path); os.IsNotExist(err) || !info.IsDir() {
		return fmt.Errorf("path %s not exist or indicate a file", path)
	}
	files, err := os.ReadDir(path)
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
	return fmt.Errorf("sync depot dir not empty, dir path %s", path)
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
	if err := CheckDepotSpace(info.DeportLocation); err != nil {
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
	delete(p.depots, name)
	return nil
}

func (p *Porter) ListAllDepot() []*DepotInfo {
	depots := []*DepotInfo{}
	for _, d := range p.depots {
		depots = append(depots, d.Depot)
	}
	return depots
}
