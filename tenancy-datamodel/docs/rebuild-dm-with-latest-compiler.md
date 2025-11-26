# Tenancy-Datamodel Rebuild Utility - Installation & Usage Guide

## What is This?

This utility automates the complete rebuild process of the tenancy-datamodel when you make changes to the nexus compiler (e.g., template files). It handles:

- ✅ Detecting uncommitted changes in compiler
- ✅ Rebuilding Docker images with local changes
- ✅ Fixing permission issues automatically
- ✅ Rebuilding the complete datamodel
- ✅ Verifying generated files

## Files Provided

1. **`rebuild-with-latest-compiler.sh`** - Main executable script
2. **`REBUILD_SCRIPT_README.md`** - Comprehensive documentation
3. **`REBUILD_QUICK_REFERENCE.sh`** - Quick command reference
4. **`INSTALL_AND_USAGE.md`** - This file

## Quick Start (3 Steps)

### Step 1: Make Changes to Compiler

```bash
cd /home/seu/workspace/orch-utils
vi nexus/compiler/pkg/generator/template/client.go.tmpl
# ... make your changes ...
```

### Step 2: Run the Rebuild Script

```bash
cd tenancy-datamodel
./rebuild-with-latest-compiler.sh
```

### Step 3: Verify Changes

```bash
grep "YourNewCode" build/nexus-client/client.go
```

That's it! The script handles everything automatically.

## Common Commands

| Task | Command |
|------|---------|
| Standard rebuild | `./rebuild-with-latest-compiler.sh` |
| Force rebuild | `./rebuild-with-latest-compiler.sh --force-rebuild` |
| Docker only | `./rebuild-with-latest-compiler.sh --docker-only` |
| Skip verification | `./rebuild-with-latest-compiler.sh --no-verify` |
| Show help | `./rebuild-with-latest-compiler.sh --help` |

## What Happens During Rebuild

The script automatically:

1. **Detects Changes**
   - Checks if compiler has uncommitted changes using `git diff`
   - Only rebuilds Docker images if changes are found (or if --force-rebuild is used)

2. **Rebuilds Compiler Images**
   - Rebuilds compiler builder image
   - Rebuilds compiler image with `INCLUDE_LOCAL_CHANGES=true`
   - This ensures Docker images include your local template changes

3. **Ensures Dependencies**
   - Checks if openapi-generator image exists
   - Builds it if needed

4. **Cleans Up Old Artifacts**
   - Removes old generated code with sudo
   - Fixes permissions on build directories
   - Prepares for fresh generation

5. **Rebuilds Datamodel**
   - Runs `make datamodel_build` with proper environment
   - Generates all client code, APIs, CRDs, etc.

6. **Verifies Success**
   - Checks for generated files
   - Confirms build completed successfully

## Key Features

### Automatic Change Detection

The script detects if you made changes in the compiler:

```bash
$ ./rebuild-with-latest-compiler.sh

[INFO] Checking for uncommitted changes in compiler...
[WARNING] Found uncommitted changes in /home/seu/workspace/orch-utils/nexus/compiler
 M pkg/generator/template/client.go.tmpl
[INFO] Rebuilding compiler images...
```

### Permission Handling

All permission issues are handled automatically with sudo:

```bash
[INFO] Removing build/nexus-client...
[INFO] Fixing permissions on build/openapi directory...
[INFO] Building datamodel...
```

### Verification

The script verifies all expected files were generated:

```bash
[SUCCESS] Found generated client.go
[SUCCESS] Found generated CRDs
[SUCCESS] Found generated APIs
[SUCCESS] Found generated OpenAPI spec
[SUCCESS] All verification checks passed (4/4)
```

## Why Not Just Use `make datamodel_build`?

The utility script provides several advantages:

| Feature | `make datamodel_build` | Script |
|---------|----------------------|--------|
| Auto-rebuild compiler | ❌ | ✅ |
| Handle permission issues | ❌ | ✅ |
| Include local changes | ❌ | ✅ |
| Change detection | ❌ | ✅ |
| File verification | ❌ | ✅ |
| Clean old artifacts | ❌ | ✅ |
| Colored output | ❌ | ✅ |

## Real-World Example

### Scenario: You update the template to add a new function

```bash
# 1. Navigate to your project and make changes
cd /home/seu/workspace/orch-utils
vi nexus/compiler/pkg/generator/template/client.go.tmpl

# 2. Add this to the template:
# func (c *Clientset) MyNewFunction() { ... }

# 3. Save and run the rebuild script
cd tenancy-datamodel
./rebuild-with-latest-compiler.sh

# Output shows:
# [INFO] Checking for uncommitted changes in compiler...
# [WARNING] Found uncommitted changes in /path/to/nexus/compiler
# [INFO] Rebuilding compiler images...
# [SUCCESS] Compiler image rebuilt successfully
# [SUCCESS] Verified: Compiler image contains latest template changes
# [INFO] Building datamodel...
# [SUCCESS] Datamodel built successfully
# [SUCCESS] All verification checks passed (4/4)

# 4. Verify your new function is in the generated code
grep "MyNewFunction" build/nexus-client/client.go
# Output: func (c *Clientset) MyNewFunction() { ... }
```

## Troubleshooting Guide

### Problem: "Command not found"

```bash
# Make sure script is executable
chmod +x rebuild-with-latest-compiler.sh

# Run from the correct directory
cd /home/seu/workspace/orch-utils/tenancy-datamodel
./rebuild-with-latest-compiler.sh
```

### Problem: "Permission denied"

The script should handle this automatically. If not:

```bash
sudo chown -R $USER:$USER build/
sudo chmod -R u+w build/
./rebuild-with-latest-compiler.sh
```

### Problem: "Docker image not found"

```bash
# Make sure Docker is running
docker ps

# If Docker is not running
sudo systemctl start docker

# Try rebuild again
./rebuild-with-latest-compiler.sh --force-rebuild
```

### Problem: Changes not appearing in generated file

```bash
# Make sure you're in the right directory
cd /home/seu/workspace/orch-utils/tenancy-datamodel

# Use force-rebuild to ensure compiler is rebuilt
./rebuild-with-latest-compiler.sh --force-rebuild

# Verify the template file was modified
cd ../nexus/compiler
git diff pkg/generator/template/client.go.tmpl

# Check the generated file
cd ../../tenancy-datamodel
grep "YourChange" build/nexus-client/client.go
```

### Problem: Build takes too long

```bash
# For quick testing of Docker images only
./rebuild-with-latest-compiler.sh --docker-only

# Skip verification checks
./rebuild-with-latest-compiler.sh --no-verify

# Both
./rebuild-with-latest-compiler.sh --docker-only --no-verify
```

## File Locations

```
/home/seu/workspace/orch-utils/
├── tenancy-datamodel/
│   ├── rebuild-with-latest-compiler.sh      ← Main script
│   ├── REBUILD_SCRIPT_README.md              ← Full documentation
│   ├── REBUILD_QUICK_REFERENCE.sh            ← Command reference
│   ├── INSTALL_AND_USAGE.md                  ← This file
│   └── build/                                 ← Generated files
│       ├── nexus-client/
│       ├── apis/
│       ├── crds/
│       └── openapi/
└── nexus/compiler/
    ├── pkg/generator/template/
    │   ├── client.go.tmpl                    ← Edit this
    │   └── ...
    └── ...
```

## Integration with Your Workflow

### Option 1: Simple Manual Workflow
```bash
# Make changes
vi ../nexus/compiler/pkg/generator/template/client.go.tmpl

# Rebuild
./rebuild-with-latest-compiler.sh
```

### Option 2: Automated Testing
```bash
# Add to your test script
cd tenancy-datamodel
./rebuild-with-latest-compiler.sh --no-verify
cd build/nexus-client
go test ./...
```

### Option 3: CI/CD Pipeline
```bash
# In your CI/CD configuration
script:
  - cd tenancy-datamodel
  - ./rebuild-with-latest-compiler.sh --docker-only
  - ./rebuild-with-latest-compiler.sh --no-verify
  - cd build && go vet ./...
```

## Performance Tips

1. **Skip verification if you're in a hurry**
   ```bash
   ./rebuild-with-latest-compiler.sh --no-verify
   ```

2. **Use --docker-only for testing**
   ```bash
   ./rebuild-with-latest-compiler.sh --docker-only
   ```

3. **Docker only rebuilds if changes detected**
   The script automatically skips rebuilding Docker images if no changes are found in the compiler directory.

4. **Subsequent builds are faster**
   The first rebuild builds Docker images (slow). Subsequent rebuilds are faster if no compiler changes are detected.

## Support & Documentation

- **Quick Reference**: `REBUILD_QUICK_REFERENCE.sh`
- **Full Documentation**: `REBUILD_SCRIPT_README.md`
- **Build Logs**: `/tmp/datamodel_build.log`

## Next Steps

1. **Try it out**
   ```bash
   cd /home/seu/workspace/orch-utils/tenancy-datamodel
   ./rebuild-with-latest-compiler.sh --help
   ```

2. **Make a test change**
   ```bash
   cd /home/seu/workspace/orch-utils
   echo "// test comment" >> nexus/compiler/pkg/generator/template/client.go.tmpl
   cd tenancy-datamodel
   ./rebuild-with-latest-compiler.sh
   ```

3. **Verify it worked**
   ```bash
   grep "test comment" build/nexus-client/client.go
   ```

4. **Read the full documentation**
   Open `REBUILD_SCRIPT_README.md` for comprehensive details

## License

SPDX-FileCopyrightText: 2025 Intel Corporation
SPDX-License-Identifier: Apache-2.0