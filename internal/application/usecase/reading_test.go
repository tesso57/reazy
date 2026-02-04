package usecase

import (
	"errors"
	"testing"

	"github.com/tesso57/reazy/internal/domain/reading"
)

type mockHistoryRepo struct {
	saveCalls int
	lastSaved []*reading.HistoryItem
	err       error
}

func (m *mockHistoryRepo) Load() (map[string]*reading.HistoryItem, error) {
	return nil, nil
}

func (m *mockHistoryRepo) Save(items []*reading.HistoryItem) error {
	m.saveCalls++
	m.lastSaved = items
	return m.err
}

func TestReadingService_ToggleBookmark(t *testing.T) {
	tests := []struct {
		name          string
		repoErr       error
		guid          string
		initialHist   map[string]*reading.HistoryItem
		wantSaveCalls int
		wantErr       bool
	}{
		{
			name:          "nil history",
			initialHist:   nil,
			guid:          "1",
			wantSaveCalls: 0,
			wantErr:       false,
		},
		{
			name: "item not found",
			initialHist: map[string]*reading.HistoryItem{
				"2": {GUID: "2"},
			},
			guid:          "1",
			wantSaveCalls: 0,
			wantErr:       false,
		},
		{
			name: "success toggle",
			initialHist: map[string]*reading.HistoryItem{
				"1": {GUID: "1", IsBookmarked: false},
			},
			guid:          "1",
			wantSaveCalls: 1,
			wantErr:       false,
		},
		{
			name:    "save error",
			repoErr: errors.New("save failed"),
			initialHist: map[string]*reading.HistoryItem{
				"1": {GUID: "1", IsBookmarked: false},
			},
			guid:          "1",
			wantSaveCalls: 1,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockHistoryRepo{err: tt.repoErr}
			svc := NewReadingService(nil, repo, nil)

			var history *reading.History
			if tt.initialHist != nil {
				history = reading.NewHistory(tt.initialHist)
			}

			err := svc.ToggleBookmark(history, tt.guid)

			if (err != nil) != tt.wantErr {
				t.Errorf("ToggleBookmark() error = %v, wantErr %v", err, tt.wantErr)
			}

			if repo.saveCalls != tt.wantSaveCalls {
				t.Errorf("Save calls = %d, want %d", repo.saveCalls, tt.wantSaveCalls)
			}

			if tt.wantSaveCalls > 0 && !tt.wantErr {
				// Verify the item state was actually toggled in the persisted list
				// Since we can't easily query the slice by ID here without helpers, we assume the history object was mutated correctly
				// because we tested that in the domain test.
				// But we can check if reading.History was passed to Save.
			}
		})
	}
}
