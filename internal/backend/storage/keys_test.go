// nolint:all // test package
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// Helper function to create a temporary KeyStore for testing.
func newTestKeyStore(t *testing.T) (*KeyStore, string) {
	t.Helper()
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "keystore.json")
	ks, err := NewKeyStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create test KeyStore: %v", err)
	}

	return ks, storePath
}

// Helper to compare slices of KeyEntry, ignoring order.
func compareKeyEntrySlices(t *testing.T, got, want []KeyEntry) bool {
	t.Helper()
	if len(got) != len(want) {
		return false
	}
	// Sort by name for consistent comparison.
	sort.Slice(got, func(i, j int) bool { return got[i].Name < got[j].Name })
	sort.Slice(want, func(i, j int) bool { return want[i].Name < want[j].Name })

	return reflect.DeepEqual(got, want)
}

func TestNewKeyStore(t *testing.T) {
	tempDir := t.TempDir()
	validPath := filepath.Join(tempDir, "keystore.json")

	// Pre-create a file with invalid JSON to test load failure.
	corruptedStorePath := filepath.Join(tempDir, "corrupted.json")
	if err := os.WriteFile(corruptedStorePath, []byte("invalid json"), 0o600); err != nil {
		t.Fatalf("Failed to write corrupted json file: %v", err)
	}

	// Pre-create a valid but empty store file.
	emptyStorePath := filepath.Join(tempDir, "empty.json")
	if err := os.WriteFile(emptyStorePath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("Failed to write empty json file: %v", err)
	}

	tests := []struct {
		name      string
		storePath string
		wantErr   bool
		setupFunc func(t *testing.T, path string) // Optional setup for specific test conditions.
	}{
		{
			name:      "valid_new_store",
			storePath: validPath,
			wantErr:   false,
		},
		{
			name:      "valid_store_parent_dir_exists",
			storePath: filepath.Join(tempDir, "existing_parent", "keystore.json"),
			setupFunc: func(t *testing.T, path string) {
				if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
					t.Fatalf("Failed to create parent dir: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name:      "load_existing_empty_store",
			storePath: emptyStorePath,
			wantErr:   false,
		},
		{
			name:      "load_corrupted_store_file",
			storePath: corruptedStorePath,
			wantErr:   true, // Expect error due to unmarshal failure.
		},
		{
			name: "unwritable_directory_for_storePath",
			// This test is platform-dependent and might require specific permissions.
			// For simplicity, we'll test the MkdirAll failure if the path is problematic.
			// On Unix-like systems, making a parent dir read-only could simulate this.
			// Here, we'll use a path that MkdirAll might struggle with if permissions were an issue.
			// A more direct test would be to make filepath.Dir(storePath) unwriteable.
			// For now, we rely on the fact that if MkdirAll fails, NewKeyStore fails.
			// Let's use a path that is a file, which should cause MkdirAll to fail for the parent.
			storePath: filepath.Join(tempDir, "file_as_parent_dir", "keystore.json"),
			setupFunc: func(t *testing.T, path string) {
				parentDirAttempt := filepath.Dir(path)
				// Create a file where a directory is expected by MkdirAll.
				if err := os.WriteFile(parentDirAttempt, []byte("I am a file"), 0o600); err != nil {
					t.Fatalf("Failed to create conflicting file: %v", err)
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc(t, tt.storePath)
			}

			ks, err := NewKeyStore(tt.storePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewKeyStore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if ks == nil {
					t.Error("NewKeyStore() returned nil KeyStore for non-error case")
					return
				}
				if ks.filePath != tt.storePath {
					t.Errorf("NewKeyStore() filePath = %v, want %v", ks.filePath, tt.storePath)
				}
				if ks.keys == nil {
					t.Error("NewKeyStore() keys map is nil")
				}
			}
		})
	}
}

func TestKeyStore_Store_Get_List_Delete(t *testing.T) {
	ks, storePath := newTestKeyStore(t)
	_ = storePath // Avoid unused variable if not directly needed for assertions here.

	entry1 := KeyEntry{
		Name:       "TestKey1",
		Type:       ZMK,
		Length:     16,
		CheckValue: "123456",
		CreatedAt:  time.Now().Truncate(time.Second),
	}
	entry2 := KeyEntry{
		Name:       "TestKey2",
		Type:       KEK,
		Length:     32,
		CheckValue: "ABCDEF",
		CreatedAt:  time.Now().Truncate(time.Second).Add(time.Minute),
	}
	entryNameless := KeyEntry{Name: "", Type: ZPK, Length: 16, CheckValue: "000000"}

	t.Run("Store_and_Get", func(t *testing.T) {
		// Store first entry.
		if err := ks.Store(entry1); err != nil {
			t.Errorf("Store() error = %v, want nil for entry1", err)
		}
		gotEntry1, exists1 := ks.Get(entry1.Name)
		if !exists1 {
			t.Errorf("Get() key %s not found after Store()", entry1.Name)
		}
		// Truncate CreatedAt for comparison as Store might set it if zero.
		entry1.CreatedAt = gotEntry1.CreatedAt // Align CreatedAt if it was set by Store.
		if !reflect.DeepEqual(gotEntry1, entry1) {
			t.Errorf("Get() got %+v, want %+v", gotEntry1, entry1)
		}

		// Store second entry.
		if err := ks.Store(entry2); err != nil {
			t.Errorf("Store() error = %v, want nil for entry2", err)
		}
		gotEntry2, exists2 := ks.Get(entry2.Name)
		if !exists2 {
			t.Errorf("Get() key %s not found after Store()", entry2.Name)
		}
		entry2.CreatedAt = gotEntry2.CreatedAt
		if !reflect.DeepEqual(gotEntry2, entry2) {
			t.Errorf("Get() got %+v, want %+v", gotEntry2, entry2)
		}

		// Get non-existent key.
		_, existsNonExistent := ks.Get("NonExistentKey")
		if existsNonExistent {
			t.Error("Get() found a non-existent key")
		}

		// Store nameless key (should error).
		if err := ks.Store(entryNameless); err == nil {
			t.Error("Store() expected error for nameless key, got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		// Ensure previous state from Store_and_Get is present.
		ks.Store(entry1) // Re-store to ensure state if tests run in parallel or are reordered.
		ks.Store(entry2)

		wantList := []KeyEntry{entry1, entry2}
		gotList := ks.List()

		if !compareKeyEntrySlices(t, gotList, wantList) {
			t.Errorf("List() got %+v, want %+v (order-independent)", gotList, wantList)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		// Ensure previous state.
		ks.Store(entry1)
		ks.Store(entry2)

		// Delete existing key.
		if err := ks.Delete(entry1.Name); err != nil {
			t.Errorf("Delete() error = %v, want nil for %s", err, entry1.Name)
		}
		_, exists := ks.Get(entry1.Name)
		if exists {
			t.Errorf("Get() found key %s after Delete()", entry1.Name)
		}

		// List should now only contain entry2.
		wantListAfterDelete := []KeyEntry{entry2}
		gotListAfterDelete := ks.List()
		if !compareKeyEntrySlices(t, gotListAfterDelete, wantListAfterDelete) {
			t.Errorf(
				"List() after delete got %+v, want %+v",
				gotListAfterDelete,
				wantListAfterDelete,
			)
		}

		// Delete non-existent key (should error).
		if err := ks.Delete("NonExistentKey"); err == nil {
			t.Error("Delete() expected error for non-existent key, got nil")
		}

		// Delete last key.
		if err := ks.Delete(entry2.Name); err != nil {
			t.Errorf("Delete() error = %v, want nil for %s", err, entry2.Name)
		}
		if len(ks.List()) != 0 {
			t.Error("List() should be empty after deleting all keys")
		}
	})

	t.Run("Persistence_load_save", func(t *testing.T) {
		ksTemp, tempStorePath := newTestKeyStore(t) // Use a fresh store for persistence test.

		entryPersist1 := KeyEntry{
			Name:       "Persist1",
			Type:       PVK,
			Length:     8,
			CheckValue: "P1P1P1",
			CreatedAt:  time.Now().Add(-time.Hour).Truncate(time.Second),
		}
		entryPersist2 := KeyEntry{
			Name:       "Persist2",
			Type:       TMK,
			Length:     24,
			CheckValue: "P2P2P2",
			CreatedAt:  time.Now().Add(-time.Minute).Truncate(time.Second),
		}

		if err := ksTemp.Store(entryPersist1); err != nil {
			t.Fatalf("Store() failed for entryPersist1: %v", err)
		}
		if err := ksTemp.Store(entryPersist2); err != nil {
			t.Fatalf("Store() failed for entryPersist2: %v", err)
		}

		// ksTemp.save() is called internally by Store. Now, create a new KeyStore from the same path.
		ksLoaded, err := NewKeyStore(tempStorePath)
		if err != nil {
			t.Fatalf("NewKeyStore() failed to load from %s: %v", tempStorePath, err)
		}

		// Check if loaded keys match.
		loadedEntry1, exists1 := ksLoaded.Get(entryPersist1.Name)
		if !exists1 {
			t.Errorf("Get() did not find %s in loaded store", entryPersist1.Name)
		}
		// Timestamps might have slight differences due to JSON marshalling if not careful.
		// Ensure CreatedAt is compared correctly, e.g., by truncating or ensuring UTC.
		// For this test, we assume CreatedAt from original entry is what gets stored and loaded.
		if !reflect.DeepEqual(loadedEntry1, entryPersist1) {
			t.Errorf("Loaded entry1 mismatch. Got %+v, want %+v", loadedEntry1, entryPersist1)
		}

		loadedEntry2, exists2 := ksLoaded.Get(entryPersist2.Name)
		if !exists2 {
			t.Errorf("Get() did not find %s in loaded store", entryPersist2.Name)
		}
		if !reflect.DeepEqual(loadedEntry2, entryPersist2) {
			t.Errorf("Loaded entry2 mismatch. Got %+v, want %+v", loadedEntry2, entryPersist2)
		}

		// Verify List.
		wantList := []KeyEntry{entryPersist1, entryPersist2}
		gotList := ksLoaded.List()
		if !compareKeyEntrySlices(t, gotList, wantList) {
			t.Errorf("List() on loaded store got %+v, want %+v", gotList, wantList)
		}

		// Test deleting from loaded store and saving again.
		if err := ksLoaded.Delete(entryPersist1.Name); err != nil {
			t.Fatalf("Delete() failed on loaded store: %v", err)
		}

		// Load again to see if delete persisted.
		ksReloaded, err := NewKeyStore(tempStorePath)
		if err != nil {
			t.Fatalf("NewKeyStore() failed to reload after delete: %v", err)
		}
		_, stillExists := ksReloaded.Get(entryPersist1.Name)
		if stillExists {
			t.Errorf("%s should have been deleted from persisted store", entryPersist1.Name)
		}
		_, existsAfterDelete := ksReloaded.Get(entryPersist2.Name)
		if !existsAfterDelete {
			t.Errorf("%s should still exist in reloaded store", entryPersist2.Name)
		}
	})
}

func TestKeyStore_load_save_edge_cases(t *testing.T) {
	t.Run("save_marshal_error", func(t *testing.T) {
		// Introduce a value that json.MarshalIndent cannot handle (e.g., a channel).
		// This is hard to do with KeyEntry struct directly unless we modify it.
		// Instead, we can simulate a marshal error by making the map itself problematic if possible,
		// or by testing this aspect at a lower level if KeyStore internals were more exposed.
		// For now, this path is hard to trigger with current KeyEntry structure.
		// A more direct way would be to mock json.MarshalIndent.
		t.Log(
			"Skipping direct test for json.MarshalIndent error due to difficulty in reliably inducing it with current KeyEntry.",
		)

		// Alternative: Test os.WriteFile error by making the file path unwriteable.
		// This is also platform-dependent.
		// Example: Make the file read-only after creation.
		ksUnwriteable, unwriteablePath := newTestKeyStore(t)
		// Store something to create the file.
		ksUnwriteable.Store(KeyEntry{Name: "dummy", Type: KEK, Length: 16, CheckValue: "dummy"})
		if err := os.Chmod(unwriteablePath, 0o400); err != nil { // Read-only.
			t.Logf("Could not set file to read-only, skipping unwriteable save test: %v", err)
			return
		}
		defer os.Chmod(unwriteablePath, 0o600) // Attempt to restore.

		err := ksUnwriteable.Store(
			KeyEntry{Name: "another", Type: ZMK, Length: 16, CheckValue: "another"},
		)
		if err == nil {
			t.Error("Store() expected error when saving to read-only file, got nil")
		} else if !os.IsPermission(err) && !strings.Contains(err.Error(), "permission denied") && !strings.Contains(err.Error(), "Access is denied.") {
			// Note: os.IsPermission might not catch all permission-related errors from WriteFile on all OSes.
			t.Errorf("Store() expected permission error, got: %v", err)
		}
	})

	t.Run("load_unmarshal_error", func(t *testing.T) {
		tempDir := t.TempDir()
		corruptedStorePath := filepath.Join(tempDir, "corrupted_load.json")
		if err := os.WriteFile(corruptedStorePath, []byte("{\"key1\": invalid_json_here}"), 0o600); err != nil {
			t.Fatalf("Failed to write corrupted json: %v", err)
		}
		_, err := NewKeyStore(corruptedStorePath)
		if err == nil {
			t.Error("NewKeyStore() expected error when loading corrupted JSON, got nil")
		} else {
			// Check if it's a json unmarshal error.
			var syntaxError *json.SyntaxError
			var unmarshalTypeError *json.UnmarshalTypeError
			if !errors.As(err, &syntaxError) && !errors.As(err, &unmarshalTypeError) && !strings.Contains(err.Error(), "failed to load keys") {
				t.Errorf("NewKeyStore() expected JSON unmarshal error, got: %v", err)
			}
		}
	})

	t.Run("load_file_not_exist", func(t *testing.T) {
		tempDir := t.TempDir()
		nonExistentPath := filepath.Join(tempDir, "ghost.json")
		ks, err := NewKeyStore(nonExistentPath)
		if err != nil {
			t.Errorf(
				"NewKeyStore() error = %v, want nil for non-existent file (should be handled gracefully)",
				err,
			)
		}
		if ks == nil {
			t.Fatal("NewKeyStore() returned nil ks for non-existent file")
		}
		if len(ks.keys) != 0 {
			t.Errorf(
				"NewKeyStore() keys map should be empty for non-existent file, got %d keys",
				len(ks.keys),
			)
		}
	})
}

func TestKeyStore_Concurrency(t *testing.T) {
	ks, _ := newTestKeyStore(t)

	numGoroutines := 50
	numOpsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(gid int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				keyName := fmt.Sprintf("ConcurrentKey_G%d_K%d", gid, j)
				entry := KeyEntry{
					Name:       keyName,
					Type:       KEK,
					Length:     16,
					CheckValue: fmt.Sprintf("cv_%d_%d", gid, j),
				}

				// Store operation.
				if err := ks.Store(entry); err != nil {
					t.Errorf("Goroutine %d: Store() error: %v", gid, err)
					return
				}

				// Get operation.
				retrievedEntry, exists := ks.Get(keyName)
				if !exists {
					t.Errorf("Goroutine %d: Get() key %s not found after Store()", gid, keyName)
					return
				}
				// Adjust CreatedAt for comparison, as Store sets it if zero.
				entry.CreatedAt = retrievedEntry.CreatedAt
				if !reflect.DeepEqual(retrievedEntry, entry) {
					t.Errorf(
						"Goroutine %d: Get() retrieved %+v, want %+v",
						gid,
						retrievedEntry,
						entry,
					)
					return
				}

				// List operation (less frequent, more to check for deadlocks).
				if j%10 == 0 {
					_ = ks.List() // Just call it to ensure no race conditions.
				}

				// Delete operation (eventually, to clean up).
				if j%2 == 0 {
					if err := ks.Delete(keyName); err != nil {
						t.Errorf("Goroutine %d: Delete() error for %s: %v", gid, keyName, err)
						return
					}
					_, stillExists := ks.Get(keyName)
					if stillExists {
						t.Errorf("Goroutine %d: Get() found key %s after Delete()", gid, keyName)
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Final check: after all operations, the number of keys should be predictable
	// if deletes are consistent. Since deletes happen for j%2 == 0, about half should be deleted.
	// However, the exact number can vary due to Store happening after Delete for the same key name in some interleavings.
	// A simpler check is that the store file can be written and re-read correctly.

	finalEntries := ks.List()
	t.Logf("Number of entries after concurrency test: %d", len(finalEntries))

	// Verify the integrity of the underlying file by reloading.
	loadedKS, err := NewKeyStore(ks.filePath)
	if err != nil {
		t.Fatalf("Failed to reload keystore after concurrency test: %v", err)
	}
	loadedEntries := loadedKS.List()

	if !compareKeyEntrySlices(t, loadedEntries, finalEntries) {
		t.Errorf(
			"Mismatch between in-memory and reloaded store after concurrency. In-memory: %d, Reloaded: %d",
			len(finalEntries),
			len(loadedEntries),
		)
	}

	// Clean up keys that might have been left if delete was not the last op for them.
	for _, e := range finalEntries {
		ks.Delete(e.Name)
	}
	if len(ks.List()) != 0 {
		t.Errorf("Cleanup at the end of concurrency test failed, %d keys remaining", len(ks.List()))
	}
}

func TestKeyStore_Store(t *testing.T) {
	validEntry := KeyEntry{Name: "TestKey1", Type: ZMK, Length: 16, CheckValue: "123CV"}
	entryWithZeroTime := KeyEntry{
		Name:       "TimeKey",
		Type:       KEK,
		Length:     8,
		CheckValue: "TimeCV",
		CreatedAt:  time.Time{},
	} // Zero time.

	tests := []struct {
		name         string
		entryToStore KeyEntry
		setupStore   func(ks *KeyStore) // Optional setup for the keystore before storing.
		wantErr      bool
		checkFunc    func(t *testing.T, ks *KeyStore, err error) // Optional check after store.
	}{
		{
			name:         "store_new_valid_entry",
			entryToStore: validEntry,
			wantErr:      false,
			checkFunc: func(t *testing.T, ks *KeyStore, err error) {
				if err != nil {
					t.Errorf("Store() unexpected error: %v", err)
					return
				}
				e, exists := ks.Get(validEntry.Name)
				if !exists {
					t.Errorf("Key %s not found after store", validEntry.Name)
				}
				if e.Name != validEntry.Name || e.Type != validEntry.Type {
					t.Errorf("Stored key mismatch. Got %+v, want (similar to) %+v", e, validEntry)
				}
				if e.CreatedAt.IsZero() {
					t.Error("CreatedAt should have been set by Store()")
				}
			},
		},
		{
			name: "update_existing_entry",
			entryToStore: KeyEntry{
				Name:       "TestKey1",
				Type:       ZPK,
				Length:     24,
				CheckValue: "UpdatedCV",
			}, // Same name as validEntry.
			setupStore: func(ks *KeyStore) {
				// Pre-store the initial version.
				if err := ks.Store(validEntry); err != nil {
					t.Fatalf("Setup: failed to store initial entry: %v", err)
				}
			},
			wantErr: false,
			checkFunc: func(t *testing.T, ks *KeyStore, err error) {
				if err != nil {
					t.Errorf("Store() unexpected error for update: %v", err)
					return
				}
				e, exists := ks.Get("TestKey1")
				if !exists {
					t.Error("Key TestKey1 not found after update")
					return
				}
				if e.Type != ZPK || e.CheckValue != "UpdatedCV" {
					t.Errorf(
						"Key TestKey1 not updated correctly. Got Type %s, CV %s",
						e.Type,
						e.CheckValue,
					)
				}
			},
		},
		{
			name:         "store_entry_with_empty_name",
			entryToStore: KeyEntry{Name: "", Type: PVK, Length: 8, CheckValue: "NoName"},
			wantErr:      true,
		},
		{
			name:         "store_entry_with_zero_created_at",
			entryToStore: entryWithZeroTime,
			wantErr:      false,
			checkFunc: func(t *testing.T, ks *KeyStore, err error) {
				if err != nil {
					t.Errorf("Store() unexpected error for zero time: %v", err)
					return
				}
				e, exists := ks.Get(entryWithZeroTime.Name)
				if !exists {
					t.Errorf(
						"Key %s not found after store (zero time test)",
						entryWithZeroTime.Name,
					)
					return
				}
				if e.CreatedAt.IsZero() {
					t.Errorf("CreatedAt was not set for entry %s", entryWithZeroTime.Name)
				}
			},
		},
		// Test for save error (e.g., read-only file).
		{
			name:         "store_with_save_error_read_only_file",
			entryToStore: KeyEntry{Name: "SaveFailKey", Type: TMK, Length: 16, CheckValue: "SFail"},
			setupStore: func(ks *KeyStore) {
				// Make the underlying file read-only.
				// First, ensure the file exists by storing a dummy entry.
				// This is a bit of a hack for a unit test, ideally save errors are mocked.
				if err := ks.Store(KeyEntry{Name: "dummySetup", Type: KEK, Length: 8, CheckValue: "dummy"}); err != nil {
					t.Logf(
						"Setup for read-only: failed to store dummy: %v. Test might not run as intended.",
						err,
					)
					// Proceeding, as Store might still fail if file doesn't exist and cannot be created.
				}
				if ks.filePath == "" {
					t.Fatal("setupStore for read-only: ks.filePath is empty")
				}
				err := os.Chmod(ks.filePath, 0o400) // Read-only.
				if err != nil {
					t.Logf(
						"Could not set file to read-only (%s): %v. This test might not be effective.",
						ks.filePath,
						err,
					)
				}
			},
			wantErr: true, // Expect an error because save will fail.
			checkFunc: func(t *testing.T, ks *KeyStore, err error) {
				if err == nil {
					t.Error("Store() expected an error due to save failure, but got nil")
				} else {
					// Check if the error is permission-related or contains "permission denied".
					// This can be OS-dependent.
					if !os.IsPermission(err) && !strings.Contains(err.Error(), "permission denied") && !strings.Contains(err.Error(), "Access is denied.") {
						t.Errorf("Store() expected a permission error, but got: %v", err)
					}
				}
				// Attempt to restore permissions for subsequent tests/cleanup.
				if ks.filePath != "" {
					os.Chmod(ks.filePath, 0o600)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks, storePath := newTestKeyStore(t) // Fresh store for each test.
			defer os.Remove(storePath)          // Clean up the test file.

			if tt.setupStore != nil {
				tt.setupStore(ks)
			}

			err := ks.Store(tt.entryToStore)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, ks, err)
			}

			// Restore permissions if changed, especially for the read-only test.
			// This is a bit broad but helps ensure cleanup.
			if strings.Contains(tt.name, "read_only_file") && ks.filePath != "" {
				os.Chmod(ks.filePath, 0o600)
			}
		})
	}
}
