package flerr_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/ttab/flerr"
)

func TestCleanerOpenFailure(t *testing.T) {
	var src ResourceSource

	src.FailForClose(0)
	src.FailForOpen(1)

	err := testCleanerLoop(t, &src, 10, func(_ int, _ *Resource, _ *Resource) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected the loop to exit with error")
	}

	// We should fail when opening the second resource (B0), then the
	// deferred close of the first resource A0 kicks in and fails and gets
	// joined with the open error.
	wantErr := `open destination: open resource with name "B0"
close source: close resource with name "A0"`

	if err.Error() != wantErr {
		t.Fatalf("unexpected error: %q", err.Error())
	}

	src.MustBeClosed(t)
}

func TestCleanerOperationAndCloseFailure(t *testing.T) {
	var src ResourceSource

	src.FailForClose(0, 1)

	err := testCleanerLoop(t, &src, 10, func(n int, _ *Resource, _ *Resource) error {
		return fmt.Errorf("operation %d failed", n)
	})
	if err == nil {
		t.Fatal("expected the loop to exit with error")
	}

	// The operation fails, then all cleanup fails as well.
	wantErr := `perform op: operation 0 failed
close source: close resource with name "A0"
close destination: close resource with name "B0"`

	if err.Error() != wantErr {
		t.Fatalf("unexpected error: %q", err.Error())
	}

	src.MustBeClosed(t)
}

func TestCleanerOperationFailure(t *testing.T) {
	var src ResourceSource

	err := testCleanerLoop(t, &src, 10, func(n int, _ *Resource, _ *Resource) error {
		if n == 2 {
			return fmt.Errorf("operation %d failed", n)
		}

		return nil
	})
	if err == nil {
		t.Fatal("expected the loop to exit with error")
	}

	// The operation fails, no other errors.
	wantErr := `perform op: operation 2 failed`

	if err.Error() != wantErr {
		t.Fatalf("unexpected error: %q", err.Error())
	}

	src.MustBeClosed(t)
}

func TestCleanerSuccess(t *testing.T) {
	var src ResourceSource

	err := testCleanerLoop(t, &src, 10, func(_ int, _, _ *Resource) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	src.MustBeClosed(t)
}

// Our test loop that simulates opening two resources and performing an
// operation on them.
func testCleanerLoop(
	t *testing.T,
	fs *ResourceSource,
	n int,
	op func(int, *Resource, *Resource) error,
) (outErr error) {
	t.Helper()

	var cleaner flerr.Cleaner

	defer cleaner.FlushTo(&outErr)

	for i := range n {
		src, err := fs.Open(fmt.Sprintf("A%d", i))
		if err != nil {
			return fmt.Errorf("open source: %w", err)
		}

		cleaner.Addf(src.Close, "close source")

		dst, err := fs.Open(fmt.Sprintf("B%d", i))
		if err != nil {
			return fmt.Errorf("open destination: %w", err)
		}

		cleaner.Addf(dst.Close, "close destination")

		err = op(i, dst, src)
		if err != nil {
			return fmt.Errorf("perform op: %w", err)
		}

		t.Logf("finished operation %d", i)

		err = cleaner.Flush()
		if err != nil {
			return err
		}
	}

	return nil
}

type Resource struct {
	err         error
	Name        string
	CloseCalled bool
}

func (r *Resource) Close() error {
	r.CloseCalled = true

	if r.err != nil {
		return r.err
	}

	return nil
}

type ResourceSource struct {
	n      int
	closeF []int
	openF  []int

	all []*Resource
}

func (rs *ResourceSource) MustBeClosed(t *testing.T) {
	t.Helper()

	var open []string

	for _, r := range rs.all {
		if r.CloseCalled {
			continue
		}

		open = append(open, r.Name)
	}

	if len(open) == 0 {
		return
	}

	t.Fatalf("resources leaked: %s", strings.Join(open, ", "))
}

func (rs *ResourceSource) FailForClose(n ...int) {
	rs.closeF = n
}

func (rs *ResourceSource) FailForOpen(n ...int) {
	rs.openF = n
}

func (rs *ResourceSource) Open(name string) (*Resource, error) {
	n := rs.n

	rs.n++

	if slices.Contains(rs.openF, n) {
		return nil, fmt.Errorf("open resource with name %q", name)
	}

	var closeErr error

	if slices.Contains(rs.closeF, n) {
		closeErr = fmt.Errorf("close resource with name %q", name)
	}

	r := Resource{
		Name: name,
		err:  closeErr,
	}

	rs.all = append(rs.all, &r)

	return &r, nil
}
