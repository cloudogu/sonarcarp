#!groovy
@Library(['github.com/cloudogu/ces-build-lib@3.1.0', 'github.com/cloudogu/dogu-build-lib@v2.6.0'])
import com.cloudogu.ces.cesbuildlib.*
@Library(['github.com/cloudogu/ces-build-lib@3.1.0', 'github.com/cloudogu/dogu-build-lib@v2.6.0'])
import com.cloudogu.ces.cesbuildlib.*
import com.cloudogu.ces.dogubuildlib.*

Git git = new Git(this, "cesmarvin")
git.committerName = 'cesmarvin'
git.committerEmail = 'cesmarvin@cloudogu.com'
gitflow = new GitFlow(this, git)
github = new GitHub(this, git)
changelog = new Changelog(this)
goVersion = "1.23.4-bullseye"
Markdown markdown = new Markdown(this, "3.11.0")
Docker docker = new Docker(this)

// Configuration of repository
repositoryOwner = "cloudogu"
repositoryName = "grafana"
project = "github.com/${repositoryOwner}/${repositoryName}"

// Configuration of branches
productionReleaseBranch = "main"

node('docker') {

    stage('Checkout') {
        checkout scm
    }

    stage('Lint') {
        Dockerfile dockerfile = new Dockerfile(this)
        dockerfile.lint()
    }

    stage('Check markdown links') {
        markdown.check()
    }

    withGolangContainer {
        stage('Build carp') {
            sh "make build-carp"
        }

        stage('Test carp') {
            sh "make carp-unit-test"
        }
    }

    stage('SonarQube') {
        projectName = 'sonarcarp'
        def scannerHome = tool name: 'sonar-scanner', type: 'hudson.plugins.sonar.SonarRunnerInstallation'
        withSonarQubeEnv {
            sh "git config 'remote.origin.fetch' '+refs/heads/*:refs/remotes/origin/*'"
            branch = env.BRANCH_NAME
            gitWithCredentials("fetch --all")

            if (branch == "main") {
                echo "This branch has been detected as the main branch."
                sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=${projectName} -Dsonar.projectName=${projectName}"
            } else if (branch == "develop") {
                echo "This branch has been detected as the develop branch."
                sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=${projectName} -Dsonar.projectName=${projectName} -Dsonar.branch.name=${branch} -Dsonar.branch.target=main  "
            } else if (env.CHANGE_TARGET) {
                echo "This branch has been detected as a pull request."
                sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=${projectName} -Dsonar.projectName=${projectName} -Dsonar.pullrequest.key=${env.CHANGE_ID} -Dsonar.pullrequest.branch=${env.CHANGE_BRANCH} -Dsonar.pullrequest.base=develop    "
            } else if (branch.startsWith("feature/")) {
                echo "This branch has been detected as a feature branch."
                sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=${projectName} -Dsonar.projectName=${projectName} -Dsonar.branch.name=${branch} -Dsonar.branch.target=develop"
            } else if (branch.startsWith("bugfix/")) {
                echo "This branch has been detected as a bugfix branch."
                sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=${projectName} -Dsonar.projectName=${projectName} -Dsonar.branch.name=${branch} -Dsonar.branch.target=develop"
            } else {
                echo "This branch has been detected as a miscellaneous branch."
                sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=${projectName} -Dsonar.projectName=${projectName} -Dsonar.branch.name=${branch} -Dsonar.branch.target=develop"
            }
        }
        timeout(time: 2, unit: 'MINUTES') { // Needed when there is no webhook for example
            def qGate = waitForQualityGate()
            if (qGate.status != 'OK') {
                unstable("Pipeline unstable due to SonarQube quality gate failure")
            }
        }
    }
}
node('vagrant') {
    timestamps {
        properties([
                // Keep only the last x builds to preserve space
                buildDiscarder(logRotator(numToKeepStr: '10')),
                // Don't run concurrent builds for a branch, because they use the same workspace directory
                disableConcurrentBuilds()
        ])

        stageAutomaticRelease()

    }
}

void withGolangContainer(Closure closure) {
    new Docker(this)
            .image("golang:${goVersion}")
            .mountJenkinsUser()
            .inside("-e ENVIRONMENT=ci") { closure.call() }
}

void gitWithCredentials(String command) {
    withCredentials([usernamePassword(credentialsId: 'cesmarvin', usernameVariable: 'GIT_AUTH_USR', passwordVariable: 'GIT_AUTH_PSW')]) {
        sh(
                script: "git -c credential.helper=\"!f() { echo username='\$GIT_AUTH_USR'; echo password='\$GIT_AUTH_PSW'; }; f\" " + command,
                returnStdout: true
        )
    }
}

void stageAutomaticRelease() {
    if (gitflow.isReleaseBranch()) {
        def releaseVersion = git.getSimpleBranchName();

        stage('Finish Release') {
            gitflow.finishRelease(releaseVersion, productionReleaseBranch)
        }

        stage('Add Github-Release') {
            github.createReleaseWithChangelog(releaseVersion, changelog, productionReleaseBranch)
        }
    }
}
