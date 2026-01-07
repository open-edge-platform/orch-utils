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
version: 26.1.0
appVersion: "26.1.0"
annotations:
  keycloak-upstream-version: "26.5.0"                    # ← Upstream Keycloak version
  keycloak-upstream-operator-image: "quay.io/keycloak/keycloak-operator:26.5.0"
  keycloak-upstream-keycloak-image: "quay.io/keycloak/keycloak:26.5.0"
  keycloak-quarkus-version: "3.27.1"                     # ← Build metadata
  keycloak-vcs-uri: "https://github.com/keycloak/keycloak.git"
  keycloak-build-timestamp: "2026-01-06 - 07:49:45 +0000"
```

#### keycloak-instance/Chart.yaml
```yaml
apiVersion: v2
name: keycloak-instance
version: 26.1.0
appVersion: "26.1.0"
annotations:
  keycloak-upstream-version: "26.5.0"                    # ← Upstream Keycloak version
  keycloak-upstream-keycloak-image: "quay.io/keycloak/keycloak:26.5.0"
  keycloak-config-cli-image: "docker.io/adorsys/keycloak-config-cli:6.4.0-26"
```

#### values.yaml (Configuration Only)

```yaml
# keycloak-operator/values.yaml
operator:
  replicas: 1
  image: "quay.io/keycloak/keycloak-operator:26.5.0"    # Used by templates
  relatedImage:
    keycloak: "quay.io/keycloak/keycloak:26.5.0"
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
        app.quarkus.io/quarkus-version: "3.27.1"      # ← Copy this
        app.quarkus.io/build-timestamp: "2026-01-06..."  # ← Copy this
      labels:
        app.kubernetes.io/version: "26.5.0"            # ← Copy this

    spec:
      containers:
      - name: keycloak-operator
        image: quay.io/keycloak/keycloak-operator:26.5.0  # ← Operator image
```

Look for environment variable:
```yaml
- name: RELATED_IMAGE_KEYCLOAK
  value: quay.io/keycloak/keycloak:26.5.0             # ← Keycloak image
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

#### 5. Verify Changes

```bash
# Lint both charts
helm lint charts/keycloak-operator charts/keycloak-instance

# Template check (verify images render correctly)
helm template test-operator charts/keycloak-operator | grep image
helm template test-instance charts/keycloak-instance | grep image
```

#### 6. Test Deployment

```bash
# Dry-run install to verify everything works
helm install --dry-run keycloak-operator charts/keycloak-operator -n orch-platform
helm install --dry-run keycloak-instance charts/keycloak-instance -n orch-platform
```

#### 7. Commit and Deploy

```bash
git add charts/keycloak-operator/Chart.yaml
git add charts/keycloak-instance/Chart.yaml
git commit -m "chore: update keycloak charts to version 26.X.0 (upstream 26.X.Y)"
```

---

## Update Checklist

When updating to a new Keycloak release, follow this checklist:

- [ ] Download new manifests from GitHub releases
- [ ] Extract version numbers from kubernetes.yml
- [ ] Update `keycloak-operator/Chart.yaml` annotations (version, images, build metadata)
- [ ] Update `keycloak-instance/Chart.yaml` annotations (version, images)
- [ ] Run `helm lint` on both charts
- [ ] Run `helm template` and verify image tags are correct
- [ ] Verify no errors in output
- [ ] **DO NOT modify values.yaml files** (versions are in Chart.yaml only)---

## Update Checklist

When updating to a new Keycloak release, follow this checklist:

- [ ] Download new manifests from GitHub releases
- [ ] Extract version numbers from kubernetes.yml
- [ ] Update `keycloak-operator/Chart.yaml` annotations (version, images, build metadata)
- [ ] Update `keycloak-instance/Chart.yaml` annotations (version, images)
- [ ] Run `helm lint` on both charts
- [ ] Run `helm template` and verify image tags are correct
- [ ] Verify no errors in output
- [ ] **DO NOT modify values.yaml files** (versions are in Chart.yaml only)
- [ ] Commit changes with clear message
- [ ] Merge PR and deploy via ArgoCD

---

## Current Versions (26.5.0)

| Component | Version | Image/Location |
|-----------|---------|---|
| **Keycloak Operator** | 26.5.0 | quay.io/keycloak/keycloak-operator:26.5.0 |
| **Keycloak Instance** | 26.5.0 | quay.io/keycloak/keycloak:26.5.0 |
| **keycloak-config-cli** | 6.4.0-26 | docker.io/adorsys/keycloak-config-cli:6.4.0-26 |
| **Quarkus** | 3.27.1 | - |
| **Chart Version (both)** | 26.1.0 | - |

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

**Last Updated**: January 6, 2026
**Current Keycloak Version**: 26.5.0
**Current Chart Version**: 26.1.0
**Status**: ✅ Production Ready
