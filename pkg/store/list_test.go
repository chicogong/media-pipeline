package store

import (
	"context"
	"testing"
	"time"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// baseTime is a fixed reference point so all timestamps are deterministic.
var baseTime = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

// makeJob returns a minimal Job with deterministic timestamps.
// id          – unique job ID
// secondsAgo  – Created = baseTime - secondsAgo seconds (larger value = older)
// updOffset   – Updated = Created + updOffset seconds
// status      – job status
func makeJob(id string, secondsAgo int, updOffset int, status schemas.JobState) *Job {
	created := baseTime.Add(-time.Duration(secondsAgo) * time.Second)
	updated := created.Add(time.Duration(updOffset) * time.Second)
	return &Job{
		JobID:   id,
		Created: created,
		Updated: updated,
		Status:  status,
	}
}

// insertJobs creates all given jobs in the store, failing fast on error.
func insertJobs(t *testing.T, s Store, jobs []*Job) {
	t.Helper()
	ctx := context.Background()
	for _, j := range jobs {
		if err := s.CreateJob(ctx, j); err != nil {
			t.Fatalf("CreateJob(%s): %v", j.JobID, err)
		}
	}
}

// jobIDs extracts the JobID slice from a result list.
func jobIDs(jobs []*Job) []string {
	ids := make([]string, len(jobs))
	for i, j := range jobs {
		ids[i] = j.JobID
	}
	return ids
}

// ---- sortJobs tests ---------------------------------------------------------

// TestListJobs_SortByCreatedAsc verifies ascending sort on Created timestamp.
func TestListJobs_SortByCreatedAsc(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	// Insert in deliberately unsorted order: job-c oldest, job-a newest
	insertJobs(t, s, []*Job{
		makeJob("job-b", 200, 0, schemas.JobStatePending),
		makeJob("job-a", 100, 0, schemas.JobStatePending),
		makeJob("job-c", 300, 0, schemas.JobStatePending),
	})

	ctx := context.Background()
	filter := &ListFilter{SortBy: "created", SortOrder: "asc"}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(result))
	}
	// ascending → oldest first: job-c (-300s) < job-b (-200s) < job-a (-100s)
	want := []string{"job-c", "job-b", "job-a"}
	got := jobIDs(result)
	for i, w := range want {
		if got[i] != w {
			t.Errorf("position %d: want %s, got %s (full order: %v)", i, w, got[i], got)
		}
	}
}

// TestListJobs_SortByCreatedDesc verifies descending sort on Created timestamp.
func TestListJobs_SortByCreatedDesc(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	insertJobs(t, s, []*Job{
		makeJob("job-b", 200, 0, schemas.JobStatePending),
		makeJob("job-a", 100, 0, schemas.JobStatePending),
		makeJob("job-c", 300, 0, schemas.JobStatePending),
	})

	ctx := context.Background()
	filter := &ListFilter{SortBy: "created", SortOrder: "desc"}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	// descending → newest first: job-a (-100s) > job-b (-200s) > job-c (-300s)
	want := []string{"job-a", "job-b", "job-c"}
	got := jobIDs(result)
	for i, w := range want {
		if got[i] != w {
			t.Errorf("position %d: want %s, got %s (full order: %v)", i, w, got[i], got)
		}
	}
}

// TestListJobs_SortByUpdatedAsc verifies ascending sort on Updated timestamp.
func TestListJobs_SortByUpdatedAsc(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	// All created at the same time; differ only by updOffset (Updated = Created + offset)
	insertJobs(t, s, []*Job{
		makeJob("job-x", 100, 50, schemas.JobStatePending), // Updated = base-100+50
		makeJob("job-y", 100, 10, schemas.JobStatePending), // Updated = base-100+10  (oldest updated)
		makeJob("job-z", 100, 90, schemas.JobStatePending), // Updated = base-100+90  (newest updated)
	})

	ctx := context.Background()
	filter := &ListFilter{SortBy: "updated", SortOrder: "asc"}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	// ascending → job-y (oldest updated) first
	want := []string{"job-y", "job-x", "job-z"}
	got := jobIDs(result)
	for i, w := range want {
		if got[i] != w {
			t.Errorf("position %d: want %s, got %s (full order: %v)", i, w, got[i], got)
		}
	}
}

// TestListJobs_SortByUpdatedDesc verifies descending sort on Updated timestamp.
func TestListJobs_SortByUpdatedDesc(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	insertJobs(t, s, []*Job{
		makeJob("job-x", 100, 50, schemas.JobStatePending),
		makeJob("job-y", 100, 10, schemas.JobStatePending),
		makeJob("job-z", 100, 90, schemas.JobStatePending),
	})

	ctx := context.Background()
	filter := &ListFilter{SortBy: "updated", SortOrder: "desc"}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	// descending → job-z (newest updated) first
	want := []string{"job-z", "job-x", "job-y"}
	got := jobIDs(result)
	for i, w := range want {
		if got[i] != w {
			t.Errorf("position %d: want %s, got %s (full order: %v)", i, w, got[i], got)
		}
	}
}

// TestListJobs_SortByStatusAsc verifies ascending lexicographic sort on Status.
func TestListJobs_SortByStatusAsc(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	// "completed" < "failed" < "pending" in ASCII order
	insertJobs(t, s, []*Job{
		makeJob("job-p", 100, 0, schemas.JobStatePending),
		makeJob("job-f", 200, 0, schemas.JobStateFailed),
		makeJob("job-c", 300, 0, schemas.JobStateCompleted),
	})

	ctx := context.Background()
	filter := &ListFilter{SortBy: "status", SortOrder: "asc"}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(result))
	}
	// Verify ascending status string order
	for i := 0; i < len(result)-1; i++ {
		if result[i].Status > result[i+1].Status {
			t.Errorf("position %d (%s) > position %d (%s): not ascending",
				i, result[i].Status, i+1, result[i+1].Status)
		}
	}
}

// TestListJobs_SortByStatusDesc verifies descending sort on Status.
func TestListJobs_SortByStatusDesc(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	insertJobs(t, s, []*Job{
		makeJob("job-p", 100, 0, schemas.JobStatePending),
		makeJob("job-f", 200, 0, schemas.JobStateFailed),
		makeJob("job-c", 300, 0, schemas.JobStateCompleted),
	})

	ctx := context.Background()
	filter := &ListFilter{SortBy: "status", SortOrder: "desc"}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	// Verify descending status string order
	for i := 0; i < len(result)-1; i++ {
		if result[i].Status < result[i+1].Status {
			t.Errorf("position %d (%s) < position %d (%s): not descending",
				i, result[i].Status, i+1, result[i+1].Status)
		}
	}
}

// TestListJobs_DefaultSort confirms that a nil SortBy falls back to
// created-descending (newest first).
func TestListJobs_DefaultSort(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	insertJobs(t, s, []*Job{
		makeJob("job-old", 300, 0, schemas.JobStatePending),
		makeJob("job-new", 100, 0, schemas.JobStatePending),
		makeJob("job-mid", 200, 0, schemas.JobStatePending),
	})

	ctx := context.Background()
	// Empty SortBy → default sort
	result, err := s.ListJobs(ctx, &ListFilter{})
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	want := []string{"job-new", "job-mid", "job-old"}
	got := jobIDs(result)
	for i, w := range want {
		if got[i] != w {
			t.Errorf("position %d: want %s, got %s (full order: %v)", i, w, got[i], got)
		}
	}
}

// TestListJobs_NilFilterSort confirms nil filter does not panic and returns all jobs.
func TestListJobs_NilFilterSort(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	insertJobs(t, s, []*Job{
		makeJob("j1", 100, 0, schemas.JobStatePending),
		makeJob("j2", 200, 0, schemas.JobStatePending),
	})

	ctx := context.Background()
	result, err := s.ListJobs(ctx, nil)
	if err != nil {
		t.Fatalf("ListJobs(nil): %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(result))
	}
}

// TestListJobs_UnknownSortBy ensures an unrecognised SortBy value does not
// panic and still returns results.
func TestListJobs_UnknownSortBy(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	insertJobs(t, s, []*Job{
		makeJob("u1", 100, 0, schemas.JobStatePending),
		makeJob("u2", 200, 0, schemas.JobStatePending),
	})

	ctx := context.Background()
	filter := &ListFilter{SortBy: "nonexistent_field", SortOrder: "asc"}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs with unknown SortBy: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(result))
	}
}

// ---- paginateJobs tests -----------------------------------------------------

// TestListJobs_LimitOnly verifies that Limit alone caps the result.
func TestListJobs_LimitOnly(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	insertJobs(t, s, []*Job{
		makeJob("p1", 100, 0, schemas.JobStatePending),
		makeJob("p2", 200, 0, schemas.JobStatePending),
		makeJob("p3", 300, 0, schemas.JobStatePending),
		makeJob("p4", 400, 0, schemas.JobStatePending),
	})

	ctx := context.Background()
	filter := &ListFilter{Limit: 2, SortBy: "created", SortOrder: "asc"}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Limit=2: expected 2 jobs, got %d", len(result))
	}
}

// TestListJobs_OffsetOnly verifies that Offset alone skips leading results.
func TestListJobs_OffsetOnly(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	insertJobs(t, s, []*Job{
		makeJob("o1", 400, 0, schemas.JobStatePending),
		makeJob("o2", 300, 0, schemas.JobStatePending),
		makeJob("o3", 200, 0, schemas.JobStatePending),
		makeJob("o4", 100, 0, schemas.JobStatePending),
	})

	ctx := context.Background()
	// Sort asc by created: o1, o2, o3, o4 — skip first 2
	filter := &ListFilter{Offset: 2, SortBy: "created", SortOrder: "asc"}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Offset=2: expected 2 jobs, got %d", len(result))
	}
	if result[0].JobID != "o3" {
		t.Errorf("first result after Offset=2 should be o3, got %s", result[0].JobID)
	}
}

// TestListJobs_LimitAndOffset verifies Limit+Offset pagination window.
func TestListJobs_LimitAndOffset(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	insertJobs(t, s, []*Job{
		makeJob("r1", 500, 0, schemas.JobStatePending),
		makeJob("r2", 400, 0, schemas.JobStatePending),
		makeJob("r3", 300, 0, schemas.JobStatePending),
		makeJob("r4", 200, 0, schemas.JobStatePending),
		makeJob("r5", 100, 0, schemas.JobStatePending),
	})

	ctx := context.Background()
	// Sort asc: r1,r2,r3,r4,r5 — skip 1, take 2 → r2,r3
	filter := &ListFilter{Offset: 1, Limit: 2, SortBy: "created", SortOrder: "asc"}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Offset=1 Limit=2: expected 2 jobs, got %d", len(result))
	}
	if result[0].JobID != "r2" || result[1].JobID != "r3" {
		t.Errorf("expected [r2, r3], got %v", jobIDs(result))
	}
}

// TestListJobs_OffsetPastEnd verifies that an Offset beyond the result count
// returns an empty (non-nil) slice.
func TestListJobs_OffsetPastEnd(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	insertJobs(t, s, []*Job{
		makeJob("e1", 100, 0, schemas.JobStatePending),
		makeJob("e2", 200, 0, schemas.JobStatePending),
	})

	ctx := context.Background()
	filter := &ListFilter{Offset: 10}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Offset past end: expected 0 jobs, got %d", len(result))
	}
}

// TestListJobs_LimitLargerThanResultSet verifies that a Limit larger than the
// total number of results simply returns all results.
func TestListJobs_LimitLargerThanResultSet(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	insertJobs(t, s, []*Job{
		makeJob("big1", 100, 0, schemas.JobStatePending),
		makeJob("big2", 200, 0, schemas.JobStatePending),
	})

	ctx := context.Background()
	filter := &ListFilter{Limit: 100}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Limit > total: expected 2 jobs, got %d", len(result))
	}
}

// ---- combined filter+sort+pagination test -----------------------------------

// TestListJobs_StatusSortPagination exercises status filtering, sorting, and
// pagination in a single call.
func TestListJobs_StatusSortPagination(t *testing.T) {
	s := NewMemoryStore()
	defer s.Close()

	// 5 pending, 3 completed — inserted in mixed order
	insertJobs(t, s, []*Job{
		makeJob("c1", 500, 0, schemas.JobStateCompleted),
		makeJob("p1", 400, 0, schemas.JobStatePending),
		makeJob("c2", 300, 0, schemas.JobStateCompleted),
		makeJob("p2", 200, 0, schemas.JobStatePending),
		makeJob("p3", 100, 0, schemas.JobStatePending),
		makeJob("c3", 600, 0, schemas.JobStateCompleted),
		makeJob("p4", 700, 0, schemas.JobStatePending),
		makeJob("p5", 800, 0, schemas.JobStatePending),
	})

	ctx := context.Background()
	// Pending only, sorted by created asc: p5,p4,p1,p2,p3 — take 2 starting at offset 1
	filter := &ListFilter{
		Status:    []schemas.JobState{schemas.JobStatePending},
		SortBy:    "created",
		SortOrder: "asc",
		Offset:    1,
		Limit:     2,
	}
	result, err := s.ListJobs(ctx, filter)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(result))
	}
	// All results must be pending
	for _, j := range result {
		if j.Status != schemas.JobStatePending {
			t.Errorf("unexpected status %s for job %s", j.Status, j.JobID)
		}
	}
	// Ascending order by Created must be preserved within the window
	if !result[0].Created.Before(result[1].Created) {
		t.Errorf("results not in ascending Created order: %s >= %s",
			result[0].Created, result[1].Created)
	}
}
