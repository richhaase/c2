package goals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/richhaase/c2/internal/atomicfile"
	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/jsonx"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
)

type Collection struct {
	Version   int    `json:"version"`
	Timezone  string `json:"timezone"`
	AccountID int64  `json:"account_id,omitempty"`
	Goals     []Goal `json:"goals"`
}

func File(p paths.DataPaths) string { return filepath.Join(p.Root, "goals.json") }

func Read(p paths.DataPaths) (*Collection, error) {
	data, err := os.ReadFile(File(p))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c Collection
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("Cannot read goals.json: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c Collection) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("Unsupported goals.json version %d.", c.Version)
	}
	if c.Timezone == "" || c.Timezone == "Local" {
		return fmt.Errorf("An explicit timezone is required, for example America/Denver or UTC.")
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		return fmt.Errorf("Invalid analysis timezone: %w", err)
	}
	if c.AccountID < 0 {
		return fmt.Errorf("Invalid store account ID.")
	}
	if c.Goals == nil {
		return fmt.Errorf("goals.json must contain a goals array (use [] for no goals).")
	}
	seen := map[string]bool{}
	for _, g := range c.Goals {
		if err := Validate(g); err != nil {
			return err
		}
		if seen[g.ID] {
			return fmt.Errorf("Duplicate goal ID %q.", g.ID)
		}
		seen[g.ID] = true
	}
	return nil
}

func Write(p paths.DataPaths, c Collection) error {
	if err := c.Validate(); err != nil {
		return err
	}
	data, err := jsonx.Indent(c)
	if err != nil {
		return err
	}
	return atomicfile.Write(File(p), append(data, '\n'), 0o644)
}

func Legacy(cfg config.Config) []Goal {
	if cfg.Goal.TargetMeters <= 0 || cfg.Goal.StartDate == "" || cfg.Goal.EndDate == "" {
		return []Goal{}
	}
	return []Goal{{ID: "legacy", Name: "Distance goal", Kind: "volume", Target: float64(cfg.Goal.TargetMeters), Equipment: "all", From: cfg.Goal.StartDate, To: cfg.Goal.EndDate, Effort: "workout"}}
}

func Load(p paths.DataPaths, cfg config.Config) ([]Goal, *time.Location, error) {
	c, err := Read(p)
	if err != nil {
		return nil, nil, err
	}
	if c == nil {
		return Legacy(cfg), time.Local, nil
	}
	loc, err := time.LoadLocation(c.Timezone)
	return c.Goals, loc, err
}

func NewCollection(cfg config.Config, zone string, workouts []models.Workout) (Collection, error) {
	c := Collection{Version: 1, Timezone: zone, Goals: Legacy(cfg)}
	if err := c.Validate(); err != nil {
		return c, err
	}
	for _, w := range workouts {
		if w.UserID <= 0 {
			continue
		}
		if c.AccountID != 0 && w.UserID != c.AccountID {
			return c, fmt.Errorf("Store contains workouts from multiple accounts; resolve before preparing it.")
		}
		c.AccountID = w.UserID
	}
	return c, nil
}

func CheckAccount(c *Collection, workouts []models.Workout, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("Cannot verify Concept2 account identity.")
	}
	if c != nil && c.AccountID != 0 && c.AccountID != userID {
		return fmt.Errorf("Authenticated account does not own this store; select the matching store before syncing.")
	}
	for _, w := range workouts {
		if w.UserID <= 0 {
			return fmt.Errorf("Workout %d has no account identity; verify its ownership before syncing.", w.ID)
		}
		if w.UserID != userID {
			return fmt.Errorf("Workout %d belongs to another account; sync stopped before changing data.", w.ID)
		}
	}
	return nil
}
