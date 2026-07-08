package services

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"backend/internal/logger"

	"go.uber.org/zap"
)

type FailureState struct {
	sync.Mutex
	APIDelay      time.Duration
	DBDelay       time.Duration
	DBFailure     bool
	VSITimeout    bool
	RandomErrors  bool
	MemoryLeak    [][]byte
	CpuBurnCancel []context.CancelFunc
}

var FailureConfig = &FailureState{
	MemoryLeak: make([][]byte, 0),
}

func (fs *FailureState) SetAPIDelay(ms int) {
	fs.Lock()
	defer fs.Unlock()
	fs.APIDelay = time.Duration(ms) * time.Millisecond
	logger.Log.Info("Simulate API Delay set", zap.Int("ms", ms))
}

func (fs *FailureState) SetDBDelay(ms int) {
	fs.Lock()
	defer fs.Unlock()
	fs.DBDelay = time.Duration(ms) * time.Millisecond
	logger.Log.Info("Simulate DB Delay set", zap.Int("ms", ms))
}

func (fs *FailureState) SetDBFailure(enable bool) {
	fs.Lock()
	defer fs.Unlock()
	fs.DBFailure = enable
	logger.Log.Info("Simulate DB Failure state changed", zap.Bool("enabled", enable))
}

func (fs *FailureState) SetVSITimeout(enable bool) {
	fs.Lock()
	defer fs.Unlock()
	fs.VSITimeout = enable
	logger.Log.Info("Simulate VSI Timeout state changed", zap.Bool("enabled", enable))
}

func (fs *FailureState) SetRandomErrors(enable bool) {
	fs.Lock()
	defer fs.Unlock()
	fs.RandomErrors = enable
	logger.Log.Info("Simulate Random HTTP Errors changed", zap.Bool("enabled", enable))
}

func (fs *FailureState) TriggerMemoryAllocation(mb int) {
	fs.Lock()
	defer fs.Unlock()
	if mb <= 0 {
		// Release memory
		fs.MemoryLeak = nil
		fs.MemoryLeak = make([][]byte, 0)
		logger.Log.Info("Released simulated memory leak pool")
		return
	}

	// Allocate megabytes
	bytesToAllocate := mb * 1024 * 1024
	leakBlock := make([]byte, bytesToAllocate)
	// Write dummy values to force OS memory allocation
	for i := 0; i < len(leakBlock); i += 4096 {
		leakBlock[i] = 1
	}
	fs.MemoryLeak = append(fs.MemoryLeak, leakBlock)
	logger.Log.Info("Allocated simulated memory leak", zap.Int("mb", mb))
}

func (fs *FailureState) SetCPUBurn(enable bool) {
	fs.Lock()
	defer fs.Unlock()

	if !enable {
		// Cancel all active CPU burn threads
		for _, cancel := range fs.CpuBurnCancel {
			cancel()
		}
		fs.CpuBurnCancel = nil
		logger.Log.Info("Stopped simulated CPU burn threads")
		return
	}

	// Start 4 CPU burning goroutines
	ctx, cancel := context.WithCancel(context.Background())
	fs.CpuBurnCancel = append(fs.CpuBurnCancel, cancel)

	for i := 0; i < 4; i++ {
		go func(workerID int) {
			logger.Log.Info("Starting CPU burn worker thread", zap.Int("worker_id", workerID))
			for {
				select {
				case <-ctx.Done():
					logger.Log.Info("Stopping CPU burn worker thread", zap.Int("worker_id", workerID))
					return
				default:
					// Infinite loop burning CPU
					_ = rand.Float64() * rand.Float64()
				}
			}
		}(i)
	}
	logger.Log.Info("Simulating CPU High Load started")
}

func (fs *FailureState) InterceptDBQuery() error {
	fs.Lock()
	defer fs.Unlock()
	
	if fs.DBDelay > 0 {
		time.Sleep(fs.DBDelay)
	}
	if fs.DBFailure {
		return errors.New("simulated database operation failure (500)")
	}
	return nil
}
