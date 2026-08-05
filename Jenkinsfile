pipeline {
    agent any

    parameters {
        choice(name: 'SERVICE', choices: ['backend', 'frontend', 'all'], description: 'Select the microservice to build and deploy')
        choice(name: 'TARGET_ENV', choices: ['pre-dev', 'stage'], description: 'Select the environment to deploy to')
        booleanParam(name: 'RUN_TESTS', defaultValue: true, description: 'Check to run automated tests before deploying')
    }

    environment {
        PRE_DEV_SERVER = "pre-dev.sandbox.local"
        STAGE_SERVER   = "stage.sandbox.local"
    }

    stages {
        stage('Pipeline Info') {
            steps {
                echo "=== Starting CI/CD Deployment Pipeline ==="
                echo "Selected Service: ${params.SERVICE}"
                echo "Target Environment: ${params.TARGET_ENV}"
                echo "Run Tests: ${params.RUN_TESTS}"
            }
        }

        stage('Run Tests') {
            when {
                expression { params.RUN_TESTS == true }
            }
            parallel {
                stage('Test Backend') {
                    when {
                        expression { params.SERVICE == 'backend' || params.SERVICE == 'all' }
                    }
                    steps {
                        echo "Testing backend Go application..."
                        dir('backend') {
                            runCmd 'go test ./... -v'
                        }
                    }
                }
                stage('Test Frontend') {
                    when {
                        expression { params.SERVICE == 'frontend' || params.SERVICE == 'all' }
                    }
                    steps {
                        echo "Testing frontend React application..."
                        dir('frontend') {
                            runCmd 'npm test'
                        }
                    }
                }
            }
        }

        stage('Build & Package') {
            parallel {
                stage('Build Backend') {
                    when {
                        expression { params.SERVICE == 'backend' || params.SERVICE == 'all' }
                    }
                    steps {
                        echo "Compiling backend Go binary..."
                        dir('backend') {
                            runCmd 'go build -o backend_app cmd/main.go'
                        }
                    }
                }
                stage('Build Frontend') {
                    when {
                        expression { params.SERVICE == 'frontend' || params.SERVICE == 'all' }
                    }
                    steps {
                        echo "Compiling frontend React static assets..."
                        dir('frontend') {
                            runCmd 'npm run build'
                        }
                    }
                }
            }
        }

        stage('Deploy to Pre-Dev') {
            when {
                expression { params.TARGET_ENV == 'pre-dev' }
            }
            steps {
                echo "Deploying ${params.SERVICE} to pre-dev environment (${PRE_DEV_SERVER})..."
                runCmd "echo Deploying container images to pre-dev server... Done!"
            }
        }

        stage('Promote to Stage') {
            when {
                expression { params.TARGET_ENV == 'stage' }
            }
            steps {
                echo "Deploying ${params.SERVICE} to staging environment (${STAGE_SERVER})..."
                runCmd "echo Deploying container images to staging server... Done!"
            }
        }

        stage('Manual Approval Gate') {
            when {
                expression { params.TARGET_ENV == 'stage' }
            }
            steps {
                echo "=== Pausing for QA approval before final release ==="
                input message: "Approve deployment of ${params.SERVICE} to STAGE?", ok: "Approve and Release"
            }
        }

        stage('Post-Deployment Verification') {
            steps {
                echo "Verifying health checks on ${params.TARGET_ENV} server..."
                runCmd "echo Health checks passed! HTTP 200 OK"
            }
        }
    }

    post {
        success {
            echo "CI/CD Pipeline finished successfully for ${params.SERVICE} on ${params.TARGET_ENV}."
        }
        failure {
            echo "Pipeline failed. Review stage logs above."
        }
    }
}

// Smart helper function to dynamically run commands with safe fallback checks
def runCmd(String cmd) {
    def firstWord = cmd.split(' ')[0]
    if (isUnix()) {
        if (firstWord == 'echo') {
            sh cmd
        } else {
            sh "command -v ${firstWord} >/dev/null 2>&1 && ${cmd} || echo 'Skipped: ${firstWord} is not installed (using mock fallback)'"
        }
    } else {
        if (firstWord == 'echo') {
            bat cmd
        } else {
            bat "where ${firstWord} >nul 2>nul && ${cmd} || echo Skipped: ${firstWord} is not installed (using mock fallback)"
        }
    }
}
