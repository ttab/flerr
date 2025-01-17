# Flush errors

Helper library that can be used to capture errors from deferred functions allows something close to a block level defer.

Example usage:

``` go
func createScriptDir(sourceDir embed.FS, scriptDir string) (outErr error) {
	err := os.MkdirAll(scriptDir, 0o700)
	if err != nil {
		return fmt.Errorf("create script directory: %w", err)
	}

	scripts, err := sourceDir.ReadDir(".")
	if err != nil {
		return fmt.Errorf("list files in source directory: %w", err)
	}

	var clean flerr.Cleaner

	// Clean up all files when the function returns, joining in any errors
	// that might result from the cleanup.
	defer clean.FlushTo(&outErr)

	for _, f := range scripts {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".py") {
			continue
		}

		src, err := pysrc.Scripts.Open(f.Name())
		if err != nil {
			return fmt.Errorf("open %q for reading: %w", f.Name(), err)
		}

		clean.Addf(src.Close, "close %q", f.Name())

		dst, err := os.Create(filepath.Join(scriptDir, f.Name()))
		if err != nil {
			return fmt.Errorf("create destination file for %q: %w", f.Name(), err)
		}

		clean.Addf(src.Close, "close %q destination", f.Name())

		_, err = io.Copy(dst, src)
		if err != nil {
			return fmt.Errorf("copy %q: %w", f.Name(), err)
		}

		// Clean up all files at the end of the loop.
		err = clean.Flush()
		if err != nil {
			return err
		}
	}

	return nil
}
```
