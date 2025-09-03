#!/bin/bash
set -euo pipefail

# Pi Controller CRD Validation Script
# This script validates the CRD definitions using kubectl

echo "🔍 Validating Pi Controller CRDs..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ kubectl is not installed or not in PATH${NC}"
    echo "Please install kubectl to validate CRDs"
    exit 1
fi

# Check if we have a Kubernetes cluster available
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${YELLOW}⚠️  No Kubernetes cluster available${NC}"
    echo "Performing offline validation only..."
    OFFLINE_ONLY=true
else
    echo -e "${GREEN}✅ Kubernetes cluster found${NC}"
    OFFLINE_ONLY=false
fi

CRD_DIR="$(dirname "$0")/../config/crd"
EXAMPLES_DIR="$(dirname "$0")/../config/examples"
ERRORS=0

echo ""
echo "📋 Validating CRD definitions..."

# Validate each CRD file syntax
for crd_file in "$CRD_DIR"/*.yaml; do
    if [[ "$(basename "$crd_file")" == "kustomization.yaml" ]] || [[ "$(basename "$crd_file")" == "README.md" ]]; then
        continue
    fi
    
    echo -n "  - $(basename "$crd_file"): "
    
    if kubectl apply --dry-run=client -f "$crd_file" &> /dev/null; then
        echo -e "${GREEN}✅ Valid${NC}"
    else
        echo -e "${RED}❌ Invalid${NC}"
        kubectl apply --dry-run=client -f "$crd_file"
        ERRORS=$((ERRORS + 1))
    fi
done

# If we have a cluster, apply CRDs and test examples
if [[ "$OFFLINE_ONLY" != "true" ]]; then
    echo ""
    echo "🚀 Testing CRD deployment..."
    
    # Apply CRDs
    echo -n "  - Applying CRDs: "
    if kubectl apply -k "$CRD_DIR" &> /dev/null; then
        echo -e "${GREEN}✅ Applied${NC}"
        
        # Wait for CRDs to be ready
        echo "  - Waiting for CRDs to be established..."
        kubectl wait --for condition=established --timeout=30s crd/gpiopins.gpio.pi-controller.io
        kubectl wait --for condition=established --timeout=30s crd/pwmcontrollers.gpio.pi-controller.io
        kubectl wait --for condition=established --timeout=30s crd/i2cdevices.gpio.pi-controller.io
        
        echo ""
        echo "📝 Testing example manifests..."
        
        # Test each example file
        for example_file in "$EXAMPLES_DIR"/*.yaml; do
            echo -n "  - $(basename "$example_file"): "
            
            if kubectl apply --dry-run=server -f "$example_file" &> /dev/null; then
                echo -e "${GREEN}✅ Valid${NC}"
            else
                echo -e "${RED}❌ Invalid${NC}"
                kubectl apply --dry-run=server -f "$example_file"
                ERRORS=$((ERRORS + 1))
            fi
        done
        
        echo ""
        echo "🧹 Cleaning up test resources..."
        kubectl delete -k "$CRD_DIR" --ignore-not-found=true &> /dev/null || true
        
    else
        echo -e "${RED}❌ Failed${NC}"
        kubectl apply -k "$CRD_DIR"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo ""
    echo "📝 Validating example manifests (offline)..."
    
    # Test example syntax only
    for example_file in "$EXAMPLES_DIR"/*.yaml; do
        echo -n "  - $(basename "$example_file"): "
        
        if kubectl apply --dry-run=client -f "$example_file" &> /dev/null; then
            echo -e "${GREEN}✅ Valid syntax${NC}"
        else
            echo -e "${RED}❌ Invalid syntax${NC}"
            kubectl apply --dry-run=client -f "$example_file"
            ERRORS=$((ERRORS + 1))
        fi
    done
fi

echo ""
echo "🔍 Checking CRD schema completeness..."

# Check for required fields in CRDs
check_crd_field() {
    local file=$1
    local field=$2
    local description=$3
    
    if grep -q "$field" "$file"; then
        echo -e "  - $description: ${GREEN}✅ Present${NC}"
    else
        echo -e "  - $description: ${RED}❌ Missing${NC}"
        ERRORS=$((ERRORS + 1))
    fi
}

# Check GPIOPin CRD
echo "  GPIOPin CRD:"
check_crd_field "$CRD_DIR/gpiopin-crd.yaml" "additionalPrinterColumns" "Custom columns"
check_crd_field "$CRD_DIR/gpiopin-crd.yaml" "shortNames" "Short names"
check_crd_field "$CRD_DIR/gpiopin-crd.yaml" "categories" "Categories"

# Check PWMController CRD  
echo "  PWMController CRD:"
check_crd_field "$CRD_DIR/pwmcontroller-crd.yaml" "additionalPrinterColumns" "Custom columns"
check_crd_field "$CRD_DIR/pwmcontroller-crd.yaml" "shortNames" "Short names"
check_crd_field "$CRD_DIR/pwmcontroller-crd.yaml" "categories" "Categories"

# Check I2CDevice CRD
echo "  I2CDevice CRD:"
check_crd_field "$CRD_DIR/i2cdevice-crd.yaml" "additionalPrinterColumns" "Custom columns"
check_crd_field "$CRD_DIR/i2cdevice-crd.yaml" "shortNames" "Short names"
check_crd_field "$CRD_DIR/i2cdevice-crd.yaml" "categories" "Categories"

echo ""
echo "📊 Validation Summary:"
if [[ $ERRORS -eq 0 ]]; then
    echo -e "${GREEN}✅ All validations passed!${NC}"
    echo "🎉 CRDs are ready for deployment"
    exit 0
else
    echo -e "${RED}❌ Found $ERRORS error(s)${NC}"
    echo "🔧 Please fix the issues above before deploying"
    exit 1
fi