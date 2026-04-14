# Keycloak Helm Charts Maintenance Guide

## Context & Architecture

### Why These Charts Exist

Keycloak does **not provide official Helm charts**. The upstream project only releases Kubernetes manifests (YAML files). Therefore:

- We created Helm charts (`keycloak-operator` and `keycloak-instance`) to wrap these manifests
- We package Keycloak deployments using standard Helm templating
- We manage version updates through centralized values configuration
- This approach provides us with flexibility, standardization, and GitOps integration

### Current Structure

```
charts/keycloak-operator/              # Operator deployment chart
  ├── Chart.yaml                       # 🎯 Chart metadata & VERSION ANNOTATIONS
  ├── values.yaml                      # Configuration (non-version values only)
  └── templates/                       # Kubernetes resources
      ├── deployment.yaml
      ├── keycloaks-crd.yaml
      └── keycloakrealmimports-crd.yaml

charts/keycloak-instance/              # Keycloak instance chart
  ├── Chart.yaml                       # 🎯 Chart metadata & VERSION ANNOTATIONS
  ├── values.yaml                      # Configuration (non-version values only)
  └── templates/                       # Kubernetes resources
      ├── keycloak.yaml
      ├── keycloak-config-cli-job.yaml
      └── other resources
```

---

## How Updates Work

### Version Management Strategy (Single Source of Truth: Chart.yaml)

**All version information is centralized in Chart.yaml annotations** to provide a single, authoritative source for all version metadata. Templates reference these annotations directly.

#### keycloak-operator/Chart.yaml
```yaml
apiVersion: v2
name: keycloak-operator
version: 26.2.0
appVersion: "26.2.0"
annotations:
  keycloak-upstream-version: "26.6.0"                    # ← Upstream Keycloak version
  keycloak-upstream-operator-image: "quay.io/keycloak/keycloak-operator:26.6.0"
  keycloak-upstream-keycloak-image: "quay.io/keycloak/keycloak:26.6.0"
  keycloak-quarkus-version: "3.33.1"                     # ← Build metadata
  keycloak-vcs-uri: "https://github.com/keycloak/keycloak.git"
  keycloak-build-timestamp: "2026-04-08 - 09:03:52 +0000"
```

#### keycloak-instance/Chart.yaml
```yaml
apiVersion: v2
name: keycloak-instance
version: 26.2.0
appVersion: "26.2.0"
annotations:
  keycloak-upstream-version: "26.6.0"                    # ← Upstream Keycloak version
  keycloak-upstream-keycloak-image: "quay.io/keycloak/keycloak:26.6.0"
  keycloak-config-cli-image: "docker.io/adorsys/keycloak-config-cli:6.4.0-26"
```

#### values.yaml (Configuration Only)

```yaml
# keycloak-operator/values.yaml
operator:
  replicas: 1
  image: '{{ index .Chart.Annotations "keycloak-upstream-operator-image" }}'
  relatedImage:
    keycloak: '{{ index .Chart.Annotations "keycloak-upstream-keycloak-image" }}'
  # ... rest is configuration (resources, security, etc)
```

### Why This Approach?

✅ **Single Source of Truth** - All versions in Chart.yaml annotations only
✅ **No Duplication** - Version appears in ONE place (Chart.yaml), not values.yaml
✅ **Standardized** - Follows Helm best practices for metadata management
✅ **Template Flexibility** - Templates can reference Chart.yaml or values.yaml as needed
✅ **Audit Trail** - Chart versioning (26.1.0, 26.2.0, etc.) tracks updates
✅ **Sustainable** - Clear separation: Chart.yaml = versions, values.yaml = configuration
✅ **Automation Ready** - Version updates could be scripted to update Chart.yaml only

---

## Updating to New Keycloak Releases

### Step-by-Step Procedure

When a new Keycloak release is available, update only **Chart.yaml** files (no values.yaml changes needed):

#### 1. Download New Manifests

```bash
# Visit: https://github.com/keycloak/keycloak/releases
# Download the Kubernetes manifests for the new version
# Extract kubernetes.yml and the two CRD files to a temporary location
cd /tmp/keycloak-NEW-VERSION
wget https://github.com/keycloak/keycloak/releases/download/VERSION/kubernetes.yml
```

#### 2. Extract Version Information

**From kubernetes.yml Deployment resource:**

Open `kubernetes.yml` and find the Deployment pod metadata:

```yaml
spec:
  template:
    metadata:
      annotations:
        app.quarkus.io/quarkus-version: "3.33.1"      # ← Copy this
        app.quarkus.io/build-timestamp: "2026-04-08..."  # ← Copy this
      labels:
        app.kubernetes.io/version: "26.6.0"            # ← Copy this

    spec:
      containers:
      - name: keycloak-operator
        image: quay.io/keycloak/keycloak-operator:26.6.0  # ← Operator image
```

Look for environment variable:
```yaml
- name: RELATED_IMAGE_KEYCLOAK
  value: quay.io/keycloak/keycloak:26.6.0             # ← Keycloak image
```

#### 3. Update Chart.yaml ONLY (keycloak-operator)

**DO NOT modify values.yaml** - only update Chart.yaml annotations:

```yaml
# keycloak-operator/Chart.yaml
apiVersion: v2
name: keycloak-operator
version: 26.X.0          # Increment minor version
appVersion: "26.X.0"     # Match version
description: Keycloak Operator Helm Chart
type: application
keywords:
  - keycloak
  - operator
annotations:
  keycloak-upstream-version: "26.X.Y"                            # NEW version
  keycloak-upstream-operator-image: "quay.io/keycloak/keycloak-operator:26.X.Y"
  keycloak-upstream-keycloak-image: "quay.io/keycloak/keycloak:26.X.Y"
  keycloak-quarkus-version: "3.Y.Z"                   # From kubernetes.yml
  keycloak-vcs-uri: "https://github.com/keycloak/keycloak.git"
  keycloak-build-timestamp: "YYYY-MM-DD - HH:MM:SS +0000"  # From kubernetes.yml
```

#### 4. Update Chart.yaml ONLY (keycloak-instance)

**Again, DO NOT modify values.yaml** - only update Chart.yaml annotations:

```yaml
# keycloak-instance/Chart.yaml
apiVersion: v2
name: keycloak-instance
version: 26.X.0          # Same version as operator
appVersion: "26.X.0"     # Match version
description: Keycloak Instance Helm Chart
type: application
keywords:
  - keycloak
  - instance
annotations:
  keycloak-upstream-version: "26.X.Y"                            # NEW version
  keycloak-upstream-keycloak-image: "quay.io/keycloak/keycloak:26.X.Y"
  keycloak-config-cli-image: "docker.io/adorsys/keycloak-config-cli:VERSION"
```

#### 5. Update Manifest Template Files (if changed)

This is the **most important step** - you must compare the downloaded manifests with the current templates and update any changes.

**5a. Update CRD Templates (if CRD definitions changed)**

CRDs define the Custom Resources that Keycloak Operator manages. They may change between releases.

```bash
# Compare old CRDs with new ones
diff -u keycloak2650/keycloaks.k8s.keycloak.org-v1.yml \
         charts/keycloak-operator/templates/keycloaks-crd.yaml

diff -u keycloak2650/keycloakrealmimports.k8s.keycloak.org-v1.yml \
         charts/keycloak-operator/templates/keycloakrealmimports-crd.yaml
```

If there are differences:

1. **Backup the old CRD files** (for reference)
2. **Update the template files** with new content:
   ```bash
   # Replace entire content with new manifest content
   cp keycloak2650/keycloaks.k8s.keycloak.org-v1.yml \
      charts/keycloak-operator/templates/keycloaks-crd.yaml
   
   cp keycloak2650/keycloakrealmimports.k8s.keycloak.org-v1.yml \
      charts/keycloak-operator/templates/keycloakrealmimports-crd.yaml
   ```

3. **Verify the files are still valid YAML**:
   ```bash
   helm lint charts/keycloak-operator
   ```

**5b. Check for Deployment Manifest Changes**

The main deployment in `kubernetes.yml` may have changes to:
- Environment variables
- Pod specifications
- Container resource requests/limits
- Security contexts
- Volume configurations

```bash
# Extract deployment section to compare
grep -A 200 "kind: Deployment" keycloak2650/kubernetes.yml > /tmp/new-deployment.yaml

# Compare with current (this is wrapped in Helm template)
# Review manually for changes
diff -u charts/keycloak-operator/templates/deployment.yaml /tmp/new-deployment.yaml
```

**Common Changes to Watch For:**
- New environment variables in `spec.template.spec.containers[].env`
- Changes to container resource limits
- New security context directives
- Changes to volume mounts
- Updates to RBAC permissions (ServiceAccount, Role, RoleBinding)

If significant changes exist:
1. Document the changes
2. Update the corresponding template files
3. Test thoroughly (see step 6)

**5c. Update RBAC Templates (if permissions changed)**

The Operator may require new RBAC permissions in new versions.

```bash
# Extract RBAC from kubernetes.yml
grep -E "kind: (ServiceAccount|Role|RoleBinding|ClusterRole|ClusterRoleBinding)" \
     keycloak2650/kubernetes.yml -A 30
```

Compare with `charts/keycloak-operator/templates/rbac-and-service.yaml` and update if needed.

---

#### 6. Verify Changes

```bash
# Lint both charts
helm lint charts/keycloak-operator charts/keycloak-instance

# Template check (verify images render correctly)
helm template test-operator charts/keycloak-operator | grep image
helm template test-instance charts/keycloak-instance | grep image
```

#### 7. Test Deployment

```bash
# Dry-run install to verify everything works
helm install --dry-run keycloak-operator charts/keycloak-operator -n orch-platform
helm install --dry-run keycloak-instance charts/keycloak-instance -n orch-platform
```

#### 8. Commit and Deploy

```bash
git add charts/keycloak-operator/Chart.yaml
git add charts/keycloak-instance/Chart.yaml
git add charts/keycloak-operator/templates/  # If manifest files changed
git add charts/keycloak-instance/templates/  # If manifest files changed
git commit -m "chore: update keycloak charts to version 26.X.0 (upstream 26.X.Y)

- Update Chart.yaml annotations for version 26.X.Y
- Update manifest templates (CRD/deployment changes from upstream)
- Update keycloak-config-cli version if available"
```

---

## Update Checklist

When updating to a new Keycloak release, follow this checklist:

- [ ] Download new manifests from GitHub releases
- [ ] Extract version numbers from kubernetes.yml
- [ ] **Compare CRD files** (keycloaks.k8s.keycloak.org-v1.yml, keycloakrealmimports.k8s.keycloak.org-v1.yml)
- [ ] **Update CRD templates if changed** (templates/keycloaks-crd.yaml, templates/keycloakrealmimports-crd.yaml)
- [ ] **Check for deployment manifest changes** (compare templates/deployment.yaml with kubernetes.yml)
- [ ] **Update deployment template if changed** (templates/deployment.yaml)
- [ ] **Check RBAC changes** (compare templates/rbac-and-service.yaml)
- [ ] **Update RBAC template if changed** (templates/rbac-and-service.yaml)
- [ ] Update `keycloak-operator/Chart.yaml` annotations (version, images, build metadata)
- [ ] Update `keycloak-instance/Chart.yaml` annotations (version, images)
- [ ] Update `keycloak-instance/templates/keycloak-config-cli-job.yaml` if needed
- [ ] Run `helm lint` on both charts (0 failures required)
- [ ] Run `helm template` and verify image tags are correct
- [ ] Verify deployment renders without errors
- [ ] **DO NOT modify values.yaml files** (versions are in Chart.yaml only)
- [ ] Commit changes with clear message
- [ ] Merge PR and deploy via ArgoCD

---

## Detailed Manifest Comparison Guide

This section provides detailed procedures for comparing and updating manifest templates when new Keycloak releases are available.

### Understanding the Manifest Structure

The charts wrap three main manifest files from upstream Keycloak:

| Manifest File | Purpose | Helm Template Location |
|---------------|---------|---|
| `kubernetes.yml` | Main Operator deployment, RBAC, Service | `templates/deployment.yaml`, `templates/rbac-and-service.yaml` |
| `keycloaks.k8s.keycloak.org-v1.yml` | CRD for Keycloak resources | `templates/keycloaks-crd.yaml` |
| `keycloakrealmimports.k8s.keycloak.org-v1.yml` | CRD for realm imports | `templates/keycloakrealmimports-crd.yaml` |

### Step-by-Step Manifest Comparison

**1. Organize Downloaded Files**

```bash
# Create a working directory for the new release
mkdir -p /tmp/keycloak-26.X.Y
cd /tmp/keycloak-26.X.Y

# Download and extract manifests
wget https://github.com/keycloak/keycloak/releases/download/26.X.Y/kubernetes.yml
wget https://github.com/keycloak/keycloak/releases/download/26.X.Y/keycloaks.k8s.keycloak.org-v1.yml
wget https://github.com/keycloak/keycloak/releases/download/26.X.Y/keycloakrealmimports.k8s.keycloak.org-v1.yml

# Copy them to your repo for easy reference
cp /tmp/keycloak-26.X.Y/* ~/workspace/orch-utils/keycloak2650-NEW/
```

**2. Compare CRD Files**

```bash
# Check if CRDs changed
diff keycloaks.k8s.keycloak.org-v1.yml \
     ~/workspace/orch-utils/charts/keycloak-operator/templates/keycloaks-crd.yaml

diff keycloakrealmimports.k8s.keycloak.org-v1.yml \
     ~/workspace/orch-utils/charts/keycloak-operator/templates/keycloakrealmimports-crd.yaml
```

**If there are changes:**
- Review the diff carefully for:
  - New spec fields
  - Changed validation rules
  - New CRD properties
  - Removed deprecated fields

```bash
# Update the template files
cp keycloaks.k8s.keycloak.org-v1.yml \
   ~/workspace/orch-utils/charts/keycloak-operator/templates/keycloaks-crd.yaml

cp keycloakrealmimports.k8s.keycloak.org-v1.yml \
   ~/workspace/orch-utils/charts/keycloak-operator/templates/keycloakrealmimports-crd.yaml

# Verify templates are valid
cd ~/workspace/orch-utils
helm lint charts/keycloak-operator
```

**3. Compare Deployment Manifest**

```bash
# Extract deployment section from new manifest
grep -n "^kind:" keycloak2650-NEW/kubernetes.yml
# Look for: kind: Deployment, kind: ServiceAccount, kind: Role, kind: RoleBinding, kind: Service

# Extract the Deployment section (adjust line numbers as needed)
sed -n '/^kind: Deployment$/,/^---$/p' keycloak2650-NEW/kubernetes.yml > /tmp/new-deployment.yaml

# Compare with current
diff charts/keycloak-operator/templates/deployment.yaml /tmp/new-deployment.yaml
```

**Key areas to check in deployment changes:**

```yaml
# ✓ Check these sections for changes:

spec:
  template:
    metadata:
      labels:           # ← New labels?
      annotations:      # ← New annotations?
    spec:
      serviceAccountName:  # ← Still the same?
      containers:
        - name: keycloak-operator
          image:              # ← Usually updated (we override via values)
          imagePullPolicy:    # ← Check for changes
          env:                # ← NEW or REMOVED env vars? (CRITICAL)
          resources:          # ← Changes to CPU/memory limits?
          ports:              # ← New ports?
          volumeMounts:       # ← New volume mounts?
          securityContext:    # ← Changes to security?
      volumes:              # ← New volumes?
      securityContext:      # ← Pod-level security changes?
```

**If deployment changed, update it:**

```bash
# Manually merge the changes into deployment.yaml
# OR if it's a complete rewrite:
sed -n '/^kind: Deployment$/,/^---$/p' keycloak2650-NEW/kubernetes.yml | \
  sed 's|image: .*|image: {{ .Values.operator.image }}|' > /tmp/deployment-new.yaml

# Review /tmp/deployment-new.yaml thoroughly before copying
cp /tmp/deployment-new.yaml charts/keycloak-operator/templates/deployment.yaml
```

**4. Compare RBAC Resources**

```bash
# Extract all RBAC resources
grep -n "^kind: \(ServiceAccount\|Role\|RoleBinding\|ClusterRole\|ClusterRoleBinding\)" \
     keycloak2650-NEW/kubernetes.yml

# Extract each section
sed -n '/^kind: ServiceAccount$/,/^---$/p' keycloak2650-NEW/kubernetes.yml > /tmp/sa-new.yaml
sed -n '/^kind: Role$/,/^---$/p' keycloak2650-NEW/kubernetes.yml > /tmp/role-new.yaml
sed -n '/^kind: RoleBinding$/,/^---$/p' keycloak2650-NEW/kubernetes.yml > /tmp/rb-new.yaml
sed -n '/^kind: ClusterRole$/,/^---$/p' keycloak2650-NEW/kubernetes.yml > /tmp/cr-new.yaml
sed -n '/^kind: ClusterRoleBinding$/,/^---$/p' keycloak2650-NEW/kubernetes.yml > /tmp/crb-new.yaml

# Compare with current rbac-and-service.yaml
diff charts/keycloak-operator/templates/rbac-and-service.yaml /tmp/sa-new.yaml
diff charts/keycloak-operator/templates/rbac-and-service.yaml /tmp/role-new.yaml
# ... etc
```

**Key things to watch for in RBAC changes:**

- ✓ New permissions in `rules` section
- ✓ New API groups being accessed
- ✓ New resource types needed
- ✓ Changes to aggregation labels
- ✓ New verbs (get, list, watch, create, update, patch, delete, etc.)

**If RBAC changed:**

```bash
# Update rbac-and-service.yaml with new permissions
# Ensure service account name stays consistent
# Test with: helm lint charts/keycloak-operator
```

**5. Verify Template Rendering**

```bash
# After updating manifests, verify they render correctly
cd ~/workspace/orch-utils

# Check that all manifests render without errors
helm template test charts/keycloak-operator 2>&1 | head -50

# Check for any YAML errors
helm template test charts/keycloak-operator > /tmp/rendered.yaml
if [ $? -eq 0 ]; then echo "✓ Templates render successfully"; fi

# Verify specific resources
helm template test charts/keycloak-operator | grep "kind:" | sort | uniq
# Should show: Deployment, Role, RoleBinding, ServiceAccount, Service, CustomResourceDefinition
```

**6. Document Changes**

Create a brief summary of manifest changes in your commit:

```bash
git log --oneline -1  # See previous format

# Your commit message should mention:
# - New env variables added/removed
# - RBAC permission changes
# - CRD field changes
# - Any security context updates
```

---

## Current Versions (26.6.0)

| Component | Version | Image/Location |
|-----------|---------|---|
| **Keycloak Operator** | 26.6.0 | quay.io/keycloak/keycloak-operator:26.6.0 |
| **Keycloak Instance** | 26.6.0 | quay.io/keycloak/keycloak:26.6.0 |
| **keycloak-config-cli** | 6.4.0-26 | docker.io/adorsys/keycloak-config-cli:6.4.0-26 |
| **Quarkus** | 3.33.1 | - |
| **Chart Version (both)** | 26.2.0 | - |

---

## Troubleshooting

### Helm Lint Fails with YAML Error

**Problem**: `error converting YAML to JSON: yaml: invalid map key`

**Solution**: Check that Chart.yaml is valid YAML with proper indentation.

### Template Rendering Shows Old Version

**Problem**: `helm template` shows old version instead of new one

**Solution**: 
1. Verify you updated `Chart.yaml` annotations correctly
2. Check templates reference the correct values or Chart.Annotations
3. Run: `helm template charts/keycloak-operator --debug | grep image`
4. Verify Chart.yaml is valid YAML with proper formatting

### Images Not Found on Registry

**Problem**: Pod fails to pull image

**Solution**:
1. Verify image tag exists: `docker pull quay.io/keycloak/keycloak-operator:VERSION`
2. Check for typos in image names in Chart.yaml
3. Verify network access to registries

### Breaking Changes in New Keycloak Version

**Problem**: Deployment fails after update

**Solution**:
1. Check Keycloak release notes for breaking changes: https://github.com/keycloak/keycloak/releases
2. Review CRD changes - may need to update templates
3. Check database migration requirements
4. Review environment variable changes

---

## Related Resources

### External
- **Keycloak Releases & Manifests**: https://github.com/keycloak/keycloak/releases
- **keycloak-config-cli Releases**: https://github.com/adorsys/keycloak-config-cli/releases
- **Quay.io Images**: https://quay.io/keycloak/keycloak
- **Helm Documentation**: https://helm.sh/docs/

### Internal
- **Operator Chart**: `charts/keycloak-operator/Chart.yaml`, `values.yaml`
- **Instance Chart**: `charts/keycloak-instance/Chart.yaml`, `values.yaml`
- **Current Manifests**: Check `/keycloak2650/` directory for reference

---

## Deployment Procedures

### Prerequisites
- Helm 3.x installed
- kubectl configured for target cluster
- ArgoCD or other GitOps tool (if using)

### Deploy to Kubernetes

```bash
# Install operator first
helm install keycloak-operator charts/keycloak-operator \
  --namespace orch-platform \
  --create-namespace

# Install instance
helm install keycloak-instance charts/keycloak-instance \
  --namespace orch-platform
```

### Using ArgoCD

Update Application manifest with new chart versions:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: platform-keycloak
spec:
  source:
    repoURL: <your-repo>
    path: charts/keycloak-operator
    targetRevision: main
    helm:
      parameters:
        - name: <param>
          value: <value>
```

### Verify Deployment

```bash
# Check operator running
kubectl get pods -n orch-platform | grep keycloak-operator

# Check instance running
kubectl get pods -n orch-platform | grep keycloak

# Check logs
kubectl logs -n orch-platform deployment/keycloak-operator
kubectl logs -n orch-platform statefulset/platform-keycloak

# Verify version
kubectl describe -n orch-platform deployment/keycloak-operator | grep image
```

---

## Maintenance Timeline

### Typical Keycloak Release Cycle
- **Frequency**: Every 1-3 months
- **Announcement**: https://github.com/keycloak/keycloak/releases
- **Update Time**: ~5 minutes (using this guide)
- **Testing Time**: 30-60 minutes
- **Deployment Time**: 15-30 minutes

### Recommended Workflow
1. **Week 0**: New release announced
2. **Week 1**: Update Chart.yaml and test in staging
3. **Week 1-2**: Validate functionality
4. **Week 2**: Deploy to production

---

## Summary

**The Goal**: Keep Keycloak charts up-to-date with upstream releases while maintaining our own Helm wrapper.

**The Process**: 
1. Download new Keycloak manifests
2. Extract version info from kubernetes.yml
3. Update Chart.yaml annotations in both charts
4. Test and deploy

**The Time**: ~5 minutes per update (Chart.yaml only - following this guide)

**The Key**: All versions are centralized in `Chart.yaml` annotations - update these annotations and templates/values automatically reference them.

---

## Questions?

Refer to relevant sections above:
- **"How do I update?"** → See "Step-by-Step Procedure"
- **"What files change?"** → See "Update Checklist"
- **"Where do I get version info?"** → See "Step 1: Download New Manifests" and "Step 2: Extract Version Information"
- **"Something broke"** → See "Troubleshooting"
- **"How do I deploy?"** → See "Deployment Procedures"

---

**Last Updated**: April 13, 2026
**Current Keycloak Version**: 26.6.0
**Current Chart Version**: 26.2.0
**Status**: ✅ Production Ready
