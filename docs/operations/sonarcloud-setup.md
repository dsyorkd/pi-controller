# SonarCloud Setup Guide

This guide provides step-by-step instructions for setting up SonarCloud analysis for the pi-controller project.

## Prerequisites

- GitHub account with admin access to the pi-controller repository
- SonarCloud account (sign up at <https://sonarcloud.io> using your GitHub account)

## Configuration Files

The following files are already configured in the repository:

- `/sonar-project.properties` - SonarCloud project configuration
- Project Key: `dsyorkd_pi-controller`
- Organization: `yorkserv`

## Manual Setup Steps

### 1. Create SonarCloud Project

1. Navigate to <https://sonarcloud.io>
2. Sign in with your GitHub account
3. Click on the "+" icon in the top right corner
4. Select "Analyze new project"
5. Choose the organization: `yorkserv`
6. Select the `dsyorkd/pi-controller` repository
7. Click "Set Up"

### 2. Generate SonarCloud Token

1. In SonarCloud, go to "My Account" > "Security"
2. Under "Tokens", enter a token name (e.g., "pi-controller-github-actions")
3. Click "Generate"
4. **IMPORTANT**: Copy the generated token immediately - it will not be shown again
5. Store this token securely - you'll need it in the next step

### 3. Configure GitHub Secrets

1. Go to your GitHub repository: <https://github.com/dsyorkd/pi-controller>
2. Navigate to "Settings" > "Secrets and variables" > "Actions"
3. Click "New repository secret"
4. Add the following secrets:

   **Secret 1: SONAR_TOKEN**
   - Name: `SONAR_TOKEN`
   - Value: [Paste the token generated in step 2]
   - Click "Add secret"

   **Secret 2: SONAR_ORGANIZATION** (optional, for reference)
   - Name: `SONAR_ORGANIZATION`
   - Value: `yorkserv`
   - Click "Add secret"

### 4. Configure Quality Gate (in SonarCloud)

1. In SonarCloud, navigate to your project
2. Go to "Project Settings" > "Quality Gates"
3. Create a new quality gate or modify the default one with these conditions:

   **Quality Gate: Pi-Controller Quality Gate**

   Conditions for new code:
   - **Coverage**: Greater than or equal to 80%
   - **Duplicated Lines (%)**: Less than 3%
   - **Maintainability Rating**: A or B (<=2)
   - **Reliability Rating**: A (=1)
   - **Security Rating**: A (=1)
   - **Security Hotspots Reviewed**: 100%

4. Assign this quality gate to your project

### 5. Enable Pull Request Decoration

1. In SonarCloud, go to "Project Settings" > "General Settings" > "Pull Requests"
2. Enable "Decorate Pull Requests"
3. Configure GitHub integration if not already done:
   - Go to "Administration" > "Integration" > "GitHub"
   - Ensure the GitHub App is installed and configured

### 6. Update Branch Settings (Optional)

1. In SonarCloud, go to "Project Settings" > "Branches and Pull Requests"
2. Set the main branch: `main`
3. Set long-lived branches: `develop`, `main`
4. Configure branch analysis settings as needed

## Verification

### Test the Integration

1. Ensure code coverage is generated:

   ```bash
   make test-coverage
   ```

2. Verify `coverage.out` file exists:

   ```bash
   ls -la coverage.out
   ```

3. Run a manual SonarCloud scan (requires sonar-scanner CLI):

   ```bash
   sonar-scanner -Dsonar.login=$SONAR_TOKEN
   ```

4. Check the SonarCloud dashboard for results

### GitHub Actions Integration

The SonarCloud scan will run automatically on:

- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches

You can verify this by:

1. Creating a test branch
2. Making a small change
3. Opening a pull request
4. Checking the "Checks" tab for SonarCloud analysis results

## Troubleshooting

### Common Issues

1. **"Not Authorized" Error**
   - Verify the `SONAR_TOKEN` secret is correctly set in GitHub
   - Regenerate the token in SonarCloud if needed

2. **Coverage Not Showing**
   - Ensure `coverage.out` is generated before the SonarCloud scan
   - Check that `sonar.go.coverage.reportPaths` is set correctly in `sonar-project.properties`

3. **Quality Gate Failing**
   - Review the specific conditions that failed in the SonarCloud dashboard
   - Adjust code or quality gate thresholds as needed

4. **Files Not Analyzed**
   - Check the exclusion patterns in `sonar-project.properties`
   - Verify source paths are correct: `cmd,internal,pkg`

### Getting Help

- SonarCloud Documentation: <https://docs.sonarcloud.io>
- Community Forum: <https://community.sonarsource.com>
- GitHub Issues: <https://github.com/dsyorkd/pi-controller/issues>

## Configuration Reference

### sonar-project.properties Overview

The `sonar-project.properties` file in the project root contains:

- **Project identification**: Key, organization, name, version
- **Source configuration**: Source and test directories
- **Coverage reports**: Path to Go coverage output
- **Exclusions**: Files and directories to exclude from analysis
  - Vendor dependencies
  - Generated code (protobuf files)
  - Test files (analyzed separately)
  - Build artifacts
  - Documentation and configuration files
- **Quality settings**: Duplication thresholds, coverage exclusions
- **SCM integration**: Git provider and repository links

### GitHub Actions Workflow

The GitHub Actions workflow should include:

```yaml
- name: Run Tests with Coverage
  run: make test-coverage

- name: SonarCloud Scan
  uses: SonarSource/sonarcloud-github-action@master
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
```

## Maintenance

### Regular Tasks

1. **Review Quality Gate**: Periodically review and adjust quality gate conditions
2. **Update Exclusions**: Add new patterns to exclusions as the codebase evolves
3. **Monitor Coverage**: Track coverage trends and address declining coverage
4. **Address Issues**: Regularly review and fix bugs, vulnerabilities, and code smells
5. **Token Rotation**: Rotate the `SONAR_TOKEN` periodically for security

### PRD Alignment

This setup aligns with PRD-12 (SonarQube Analysis) requirements:

- ✓ Automated code quality metrics
- ✓ Security vulnerability scanning
- ✓ Technical debt measurement
- ✓ Code coverage tracking (>80% target)
- ✓ Quality gates for PR reviews
- ✓ Maintainability rating A or B
- ✓ Security rating A
- ✓ Duplicated code <3%

## Next Steps

After completing the setup:

1. Review the initial scan results in SonarCloud
2. Address critical and blocker issues (PRD Task 3.1)
3. Configure quality gates per PRD requirements (PRD Task 3.2)
4. Integrate SonarCloud status into PR review process
5. Train team members on quality standards and SonarCloud usage

## References

- [SonarCloud Documentation](https://docs.sonarcloud.io)
- [SonarCloud Go Analysis](https://docs.sonarcloud.io/advanced-setup/languages/go/)
- [GitHub Actions Integration](https://github.com/SonarSource/sonarcloud-github-action)
- [Quality Gates](https://docs.sonarcloud.io/improving/quality-gates/)
- PRD: `.taskmaster/docs/prd-12-sonarqube-analysis.txt`
