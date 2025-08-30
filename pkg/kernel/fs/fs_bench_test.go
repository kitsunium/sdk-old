package fs_test

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/kitsunium/sdk/pkg/kernel/fs"
)

// BenchmarkArchive contains benchmarks for archive operations
func BenchmarkArchive(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench_archive")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files for benchmarking
	testFiles := []string{}
	for i := 0; i < 100; i++ {
		fileName := filepath.Join(tmpDir, "testfile_"+string(rune('0'+i%10))+".txt")
		err := os.WriteFile(fileName, []byte("test content for benchmarking archive operations"), 0644)
		if err != nil {
			b.Fatal(err)
		}
		testFiles = append(testFiles, fileName)
	}

	b.Run("NewArchive", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			archive := fs.NewArchive(filepath.Join(tmpDir, "bench.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			_ = archive
		}
	})

	b.Run("AddFile_Sequential", func(b *testing.B) {
		archive := fs.NewArchive(filepath.Join(tmpDir, "bench_seq.zip"), fs.ArchiveOptions{
			ArchiveType:     fs.ZIP,
			CompressionType: fs.GZIP,
		})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := archive.AddFile(testFiles[i%len(testFiles)])
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("AddFile_Parallel", func(b *testing.B) {
		archive := fs.NewArchive(filepath.Join(tmpDir, "bench_par.zip"), fs.ArchiveOptions{
			ArchiveType:     fs.ZIP,
			CompressionType: fs.GZIP,
		})

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				err := archive.AddFile(testFiles[i%len(testFiles)])
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})

	b.Run("Compress_Small", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			archive := fs.NewArchive(filepath.Join(tmpDir, "compress_small.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			_ = archive.AddFile(testFiles[0])
			err := archive.Compress()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Compress_Large", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			archive := fs.NewArchive(filepath.Join(tmpDir, "compress_large.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			for _, file := range testFiles[:10] { // Add 10 files
				_ = archive.AddFile(file)
			}
			err := archive.Compress()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDirectory contains benchmarks for directory operations
func BenchmarkDirectory(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench_directory")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested directory structure
	for i := 0; i < 10; i++ {
		dirPath := filepath.Join(tmpDir, "dir_"+string(rune('0'+i)))
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			b.Fatal(err)
		}

		// Add files to each directory
		for j := 0; j < 5; j++ {
			fileName := filepath.Join(dirPath, "file_"+string(rune('0'+j))+".txt")
			err := os.WriteFile(fileName, []byte("benchmark content"), 0644)
			if err != nil {
				b.Fatal(err)
			}
		}
	}

	b.Run("NewDirectory", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dir, err := fs.NewDirectory(fs.Option{Path: tmpDir})
			if err != nil {
				b.Fatal(err)
			}
			_ = dir
		}
	})

	b.Run("Create", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testDirPath := filepath.Join(tmpDir, "create_bench_"+string(rune('0'+i%10)))
			dir, err := fs.NewDirectory(fs.Option{Path: testDirPath})
			if err != nil {
				b.Fatal(err)
			}

			createdDir, err := dir.Create()
			if err != nil {
				b.Fatal(err)
			}
			_ = createdDir
		}
	})

	b.Run("List", func(b *testing.B) {
		dir, err := fs.NewDirectory(fs.Option{Path: tmpDir})
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			files, dirs, err := dir.List()
			if err != nil {
				b.Fatal(err)
			}
			_ = files
			_ = dirs
		}
	})

	b.Run("List_Parallel", func(b *testing.B) {
		dir, err := fs.NewDirectory(fs.Option{Path: tmpDir})
		if err != nil {
			b.Fatal(err)
		}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				files, dirs, err := dir.List()
				if err != nil {
					b.Fatal(err)
				}
				_ = files
				_ = dirs
			}
		})
	})

	b.Run("Size", func(b *testing.B) {
		dir, err := fs.NewDirectory(fs.Option{Path: tmpDir})
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			size := dir.Size()
			_ = size
		}
	})

	b.Run("Size_Parallel", func(b *testing.B) {
		dir, err := fs.NewDirectory(fs.Option{Path: tmpDir})
		if err != nil {
			b.Fatal(err)
		}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				size := dir.Size()
				_ = size
			}
		})
	})

	b.Run("Has", func(b *testing.B) {
		dir, err := fs.NewDirectory(fs.Option{Path: tmpDir})
		if err != nil {
			b.Fatal(err)
		}
		testPath := filepath.Join(tmpDir, "dir_0", "file_0.txt")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			has := dir.Has(testPath)
			_ = has
		}
	})

	b.Run("Exists", func(b *testing.B) {
		dir, err := fs.NewDirectory(fs.Option{Path: tmpDir})
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			exists := dir.Exists()
			_ = exists
		}
	})
}

// BenchmarkFile contains benchmarks for file operations
func BenchmarkFile(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench_file")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with different sizes
	smallFile := filepath.Join(tmpDir, "small.txt")
	mediumFile := filepath.Join(tmpDir, "medium.txt")
	largeFile := filepath.Join(tmpDir, "large.txt")

	smallContent := make([]byte, 1024)      // 1KB
	mediumContent := make([]byte, 1024*100) // 100KB
	largeContent := make([]byte, 1024*1024) // 1MB

	for i := range smallContent {
		smallContent[i] = byte(i % 256)
	}
	for i := range mediumContent {
		mediumContent[i] = byte(i % 256)
	}
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	err = os.WriteFile(smallFile, smallContent, 0644)
	if err != nil {
		b.Fatal(err)
	}
	err = os.WriteFile(mediumFile, mediumContent, 0644)
	if err != nil {
		b.Fatal(err)
	}
	err = os.WriteFile(largeFile, largeContent, 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("NewFile", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			file, err := fs.NewFile(fs.Option{Path: smallFile})
			if err != nil {
				b.Fatal(err)
			}
			_ = file
		}
	})

	b.Run("Read_Small", func(b *testing.B) {
		file, err := fs.NewFile(fs.Option{Path: smallFile})
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			content, err := file.Read()
			if err != nil {
				b.Fatal(err)
			}
			_ = content
		}
	})

	b.Run("Read_Medium", func(b *testing.B) {
		file, err := fs.NewFile(fs.Option{Path: mediumFile})
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			content, err := file.Read()
			if err != nil {
				b.Fatal(err)
			}
			_ = content
		}
	})

	b.Run("Read_Large", func(b *testing.B) {
		file, err := fs.NewFile(fs.Option{Path: largeFile})
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			content, err := file.Read()
			if err != nil {
				b.Fatal(err)
			}
			_ = content
		}
	})

	b.Run("Read_Parallel_Small", func(b *testing.B) {
		file, err := fs.NewFile(fs.Option{Path: smallFile})
		if err != nil {
			b.Fatal(err)
		}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				content, err := file.Read()
				if err != nil {
					b.Fatal(err)
				}
				_ = content
			}
		})
	})

	b.Run("Write_Small", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			writeFile := filepath.Join(tmpDir, "write_small_"+string(rune('0'+i%10))+".txt")
			file, err := fs.NewFile(fs.Option{Path: writeFile})
			if err != nil {
				b.Fatal(err)
			}

			_, err = file.Write(smallContent[:100]) // 100 bytes
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Write_Medium", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			writeFile := filepath.Join(tmpDir, "write_medium_"+string(rune('0'+i%10))+".txt")
			file, err := fs.NewFile(fs.Option{Path: writeFile})
			if err != nil {
				b.Fatal(err)
			}

			_, err = file.Write(mediumContent[:10000]) // 10KB
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Copy_Small", func(b *testing.B) {
		file, err := fs.NewFile(fs.Option{Path: smallFile})
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dstFile := filepath.Join(tmpDir, "copy_small_"+string(rune('0'+i%10))+".txt")
			err := file.Copy(dstFile)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Copy_Medium", func(b *testing.B) {
		file, err := fs.NewFile(fs.Option{Path: mediumFile})
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dstFile := filepath.Join(tmpDir, "copy_medium_"+string(rune('0'+i%10))+".txt")
			err := file.Copy(dstFile)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Copy_Large", func(b *testing.B) {
		file, err := fs.NewFile(fs.Option{Path: largeFile})
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dstFile := filepath.Join(tmpDir, "copy_large_"+string(rune('0'+i%10))+".txt")
			err := file.Copy(dstFile)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Copy_Parallel", func(b *testing.B) {
		file, err := fs.NewFile(fs.Option{Path: smallFile})
		if err != nil {
			b.Fatal(err)
		}

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				dstFile := filepath.Join(tmpDir, "copy_par_"+string(rune('0'+i%10))+".txt")
				err := file.Copy(dstFile)
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})

	b.Run("Create", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			createFile := filepath.Join(tmpDir, "create_"+string(rune('0'+i%10))+".txt")
			file, err := fs.NewFile(fs.Option{Path: createFile})
			if err != nil {
				b.Fatal(err)
			}

			createdFile, err := file.Create()
			if err != nil {
				b.Fatal(err)
			}
			_ = createdFile
		}
	})

	b.Run("IsDotFile", func(b *testing.B) {
		file, err := fs.NewFile(fs.Option{Path: smallFile})
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			isDot := file.IsDotFile()
			_ = isDot
		}
	})

	b.Run("Size", func(b *testing.B) {
		file, err := fs.NewFile(fs.Option{Path: smallFile})
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			size := file.Size()
			_ = size
		}
	})

	b.Run("Exists", func(b *testing.B) {
		file, err := fs.NewFile(fs.Option{Path: smallFile})
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			exists := file.Exists()
			_ = exists
		}
	})
}

// BenchmarkStats contains benchmarks for stats operations
func BenchmarkStats(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench_stats")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "stats_test.txt")
	err = os.WriteFile(testFile, []byte("stats benchmarking content"), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("NewStats", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			stats := fs.NewStats(testFile)
			_ = stats
		}
	})

	b.Run("Refresh", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := stats.Refresh()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Refresh_Parallel", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := stats.Refresh()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})

	b.Run("Owner", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			owner := stats.Owner()
			_ = owner
		}
	})

	b.Run("Owner_Parallel", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				owner := stats.Owner()
				_ = owner
			}
		})
	})

	b.Run("Group", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			group := stats.Group()
			_ = group
		}
	})

	b.Run("Other", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			other := stats.Other()
			_ = other
		}
	})

	b.Run("HasPermissions", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			has := stats.HasPermissions(0644)
			_ = has
		}
	})

	b.Run("IsReadable", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			readable := stats.IsReadable(1000, 1000)
			_ = readable
		}
	})

	b.Run("IsWritable", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			writable := stats.IsWritable(1000, 1000)
			_ = writable
		}
	})

	b.Run("IsExecutable", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			executable := stats.IsExecutable(1000, 1000)
			_ = executable
		}
	})

	b.Run("IsFile", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			isFile := stats.IsFile()
			_ = isFile
		}
	})

	b.Run("IsDirectory", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			isDir := stats.IsDirectory()
			_ = isDir
		}
	})

	b.Run("Exists", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			exists := stats.Exists()
			_ = exists
		}
	})

	b.Run("Chmod", func(b *testing.B) {
		stats := fs.NewStats(testFile)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := stats.Chmod(0644)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMultiCore contains benchmarks designed to test multi-core performance
func BenchmarkMultiCore(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench_multicore")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files for multi-core benchmarking
	numFiles := runtime.NumCPU() * 10
	testFiles := make([]string, numFiles)
	testContent := make([]byte, 1024*10) // 10KB per file
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}

	for i := 0; i < numFiles; i++ {
		fileName := filepath.Join(tmpDir, "multicore_"+string(rune('0'+i%10))+".txt")
		err := os.WriteFile(fileName, testContent, 0644)
		if err != nil {
			b.Fatal(err)
		}
		testFiles[i] = fileName
	}

	b.Run("File_Read_MultiCore", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				file, err := fs.NewFile(fs.Option{Path: testFiles[i%len(testFiles)]})
				if err != nil {
					b.Fatal(err)
				}

				content, err := file.Read()
				if err != nil {
					b.Fatal(err)
				}
				_ = content
				i++
			}
		})
	})

	b.Run("File_Copy_MultiCore", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				file, err := fs.NewFile(fs.Option{Path: testFiles[i%len(testFiles)]})
				if err != nil {
					b.Fatal(err)
				}

				dstFile := filepath.Join(tmpDir, "copy_mc_"+string(rune('0'+i%10))+".txt")
				err = file.Copy(dstFile)
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})

	b.Run("Stats_Refresh_MultiCore", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				stats := fs.NewStats(testFiles[i%len(testFiles)])
				err := stats.Refresh()
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})

	b.Run("Directory_List_MultiCore", func(b *testing.B) {
		// Create subdirectories for parallel listing
		subDirs := make([]string, runtime.NumCPU())
		for i := 0; i < len(subDirs); i++ {
			subDir := filepath.Join(tmpDir, "subdir_"+string(rune('0'+i)))
			err := os.MkdirAll(subDir, 0755)
			if err != nil {
				b.Fatal(err)
			}

			// Add some files to each subdirectory
			for j := 0; j < 5; j++ {
				fileName := filepath.Join(subDir, "file_"+string(rune('0'+j))+".txt")
				err := os.WriteFile(fileName, testContent[:100], 0644)
				if err != nil {
					b.Fatal(err)
				}
			}
			subDirs[i] = subDir
		}

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				dir, err := fs.NewDirectory(fs.Option{Path: subDirs[i%len(subDirs)]})
				if err != nil {
					b.Fatal(err)
				}

				files, dirs, err := dir.List()
				if err != nil {
					b.Fatal(err)
				}
				_ = files
				_ = dirs
				i++
			}
		})
	})

	b.Run("Mixed_Operations_MultiCore", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				testFile := testFiles[i%len(testFiles)]

				// Mix of operations
				switch i % 4 {
				case 0:
					// File read
					file, err := fs.NewFile(fs.Option{Path: testFile})
					if err != nil {
						b.Fatal(err)
					}
					_, err = file.Read()
					if err != nil {
						b.Fatal(err)
					}

				case 1:
					// Stats refresh
					stats := fs.NewStats(testFile)
					err := stats.Refresh()
					if err != nil {
						b.Fatal(err)
					}

				case 2:
					// File copy
					file, err := fs.NewFile(fs.Option{Path: testFile})
					if err != nil {
						b.Fatal(err)
					}
					dstFile := filepath.Join(tmpDir, "mixed_"+string(rune('0'+i%10))+".txt")
					err = file.Copy(dstFile)
					if err != nil {
						b.Fatal(err)
					}

				case 3:
					// Directory operations
					dir, err := fs.NewDirectory(fs.Option{Path: tmpDir})
					if err != nil {
						b.Fatal(err)
					}
					has := dir.Has(testFile)
					_ = has
				}
				i++
			}
		})
	})
}

// BenchmarkConcurrencyStress tests the package under high concurrency stress
func BenchmarkConcurrencyStress(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench_stress")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create many test files
	numFiles := 1000
	testFiles := make([]string, numFiles)
	testContent := []byte("stress test content for concurrency benchmarking")

	for i := 0; i < numFiles; i++ {
		fileName := filepath.Join(tmpDir, "stress_"+string(rune('0'+i%10))+".txt")
		err := os.WriteFile(fileName, testContent, 0644)
		if err != nil {
			b.Fatal(err)
		}
		testFiles[i] = fileName
	}

	b.Run("High_Concurrency_File_Operations", func(b *testing.B) {
		var wg sync.WaitGroup
		numGoroutines := runtime.NumCPU() * 4

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(numGoroutines)

			for g := 0; g < numGoroutines; g++ {
				go func(goroutineID int) {
					defer wg.Done()

					for j := 0; j < 10; j++ {
						fileIdx := (goroutineID*10 + j) % len(testFiles)
						testFile := testFiles[fileIdx]

						file, err := fs.NewFile(fs.Option{Path: testFile})
						if err != nil {
							b.Errorf("Failed to create file: %v", err)
							return
						}

						// Perform various operations
						_, err = file.Read()
						if err != nil {
							b.Errorf("Failed to read file: %v", err)
							return
						}

						exists := file.Exists()
						_ = exists

						size := file.Size()
						_ = size
					}
				}(g)
			}

			wg.Wait()
		}
	})

	b.Run("High_Concurrency_Stats_Operations", func(b *testing.B) {
		var wg sync.WaitGroup
		numGoroutines := runtime.NumCPU() * 4

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(numGoroutines)

			for g := 0; g < numGoroutines; g++ {
				go func(goroutineID int) {
					defer wg.Done()

					for j := 0; j < 10; j++ {
						fileIdx := (goroutineID*10 + j) % len(testFiles)
						testFile := testFiles[fileIdx]

						stats := fs.NewStats(testFile)

						err := stats.Refresh()
						if err != nil {
							b.Errorf("Failed to refresh stats: %v", err)
							return
						}

						owner := stats.Owner()
						_ = owner

						group := stats.Group()
						_ = group

						exists := stats.Exists()
						_ = exists
					}
				}(g)
			}

			wg.Wait()
		}
	})
}

// BenchmarkMemoryUsage measures memory efficiency of the fs package
func BenchmarkMemoryUsage(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench_memory")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "memory_test.txt")
	err = os.WriteFile(testFile, []byte("memory usage benchmark"), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("File_Object_Creation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			file, err := fs.NewFile(fs.Option{Path: testFile})
			if err != nil {
				b.Fatal(err)
			}
			_ = file
		}
	})

	b.Run("Stats_Object_Creation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			stats := fs.NewStats(testFile)
			_ = stats
		}
	})

	b.Run("Directory_Object_Creation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			dir, err := fs.NewDirectory(fs.Option{Path: tmpDir})
			if err != nil {
				b.Fatal(err)
			}
			_ = dir
		}
	})

	b.Run("Archive_Object_Creation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			archive := fs.NewArchive(filepath.Join(tmpDir, "memory.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			_ = archive
		}
	})
}
