package upgrade

import (
	"context"
	"log"
	"os"
	"sync"
	"time"
)

var computeFileHashFunc = ComputeFileHash

// Config holds all settings for the upgrade manager.
type Config struct {
	// ReleaseURL is the GitHub Releases API endpoint.
	// Defaults to DefaultReleaseURL.
	ReleaseURL string

	// CheckInterval is how often to poll for new releases.
	// Defaults to 1 hour.
	CheckInterval time.Duration

	// AutoApply: if true the manager applies a newer release immediately after
	// downloading and verifying it.  If false it only logs a notice.
	AutoApply bool

	// AllowDowngrade: if true even older releases are applied (for rollback).
	AllowDowngrade bool
}

func (c *Config) defaults() {
	if c.ReleaseURL == "" {
		c.ReleaseURL = DefaultReleaseURL
	}
	if c.CheckInterval <= 0 {
		c.CheckInterval = 1 * time.Hour
	}
}

// Manager orchestrates the full upgrade lifecycle:
//   - Periodic release checks against GitHub
//   - Download + SHA-3 verification
//   - Binary replacement via Apply
//   - On-chain upgrade scheduling via Scheduler
type Manager struct {
	cfg       Config
	scheduler *Scheduler
	once      sync.Once // ensures we only apply once per session
}

// NewManager creates an upgrade Manager.
// scheduler may be nil to disable on-chain upgrade watching.
func NewManager(cfg Config, scheduler *Scheduler) *Manager {
	cfg.defaults()
	return &Manager{cfg: cfg, scheduler: scheduler}
}

// Run starts the periodic release-check loop; blocks until ctx is cancelled.
func (m *Manager) Run(ctx context.Context) {
	// Clean up any leftover .old binary from a previous upgrade.
	cleanOldBinary()

	log.Printf("[upgrade] manager started (current=%s, check_interval=%s, auto_apply=%v)",
		Current(), m.cfg.CheckInterval, m.cfg.AutoApply)

	ticker := time.NewTicker(m.cfg.CheckInterval)
	defer ticker.Stop()

	// Run one check immediately on startup.
	m.check(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[upgrade] manager stopped")
			return
		case <-ticker.C:
			m.check(ctx)
		}
	}
}

// check fetches the latest release and, if appropriate, downloads and applies it.
func (m *Manager) check(ctx context.Context) {
	rel, err := FetchLatestRelease(ctx, m.cfg.ReleaseURL)
	if err != nil {
		log.Printf("[upgrade] release check failed: %v", err)
		return
	}

	current := Current()
	isNewer := rel.Version.After(current)
	isOlder := current.After(rel.Version)

	switch {
	case rel.Version.Equal(current):
		log.Printf("[upgrade] already on latest release %s", current)
		return
	case isNewer:
		log.Printf("[upgrade] new release available: %s → %s (%s)",
			current, rel.Version, rel.ReleaseURL)
	case isOlder && m.cfg.AllowDowngrade:
		log.Printf("[upgrade] downgrade available: %s → %s", current, rel.Version)
	case isOlder:
		log.Printf("[upgrade] running newer version %s than published %s — skipping",
			current, rel.Version)
		return
	}

	if !m.cfg.AutoApply {
		log.Printf("[upgrade] auto-apply disabled; manual upgrade required to %s", rel.Version)
		return
	}

	m.applyRelease(ctx, rel)
}

// applyRelease downloads, verifies, and applies a release.
// Protected by sync.Once so only one upgrade runs per process lifetime.
func (m *Manager) applyRelease(ctx context.Context, rel *Release) {
	m.once.Do(func() {
		log.Printf("[upgrade] downloading %s from %s", rel.Version, rel.DownloadURL)

		tmpPath, err := Download(ctx, rel)
		if err != nil {
			log.Printf("[upgrade] download failed: %v", err)
			return
		}

		// Re-verify on-disk checksum before replacing the binary (defence-in-depth).
		if rel.Checksum != "" {
			onDisk, hashErr := computeFileHashFunc(tmpPath)
			if hashErr != nil {
				log.Printf("[upgrade] on-disk hash error: %v — aborting", hashErr)
				_ = os.Remove(tmpPath)
				return
			}
			if onDisk != rel.Checksum {
				log.Printf("[upgrade] on-disk checksum mismatch (want=%s got=%s) — aborting",
					rel.Checksum, onDisk)
				_ = os.Remove(tmpPath)
				return
			}
		}
		log.Printf("[upgrade] verified %s (SHA-3-256 ✓); applying…", rel.Version)

		// Apply does not return on success (POSIX exec / Windows exit).
		if err := Apply(tmpPath); err != nil {
			log.Printf("[upgrade] apply failed: %v", err)
		}
	})
}

// OnBlock forwards the block height to the Scheduler (if configured).
func (m *Manager) OnBlock(height uint64) {
	if m.scheduler != nil {
		m.scheduler.OnBlock(height)
	}
}

// AddProposal records an on-chain upgrade proposal received from a peer.
// Called by the P2P layer when a peer broadcasts an upgrade proposal message;
// also exercised directly by the scheduler tests.
func (m *Manager) AddProposal(p *Proposal) (bool, error) {
	if m.scheduler == nil {
		return false, nil
	}
	return m.scheduler.AddProposal(p)
}
