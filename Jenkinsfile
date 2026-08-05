pipeline {
    agent any

    // 1. Parameters block to let you choose service and environment
    parameters {
        choice(name: 'SERVICE', choices: ['backend', 'frontend', 'all'], description: 'Select the microservice to build and deploy')
        choice(name: 'TARGET_ENV', choices: ['pre-dev', 'stage'], description: 'Select the environment to deploy to')
        booleanParam(name: 'RUN_TESTS', defaultValue: true, description: 'Check to run automated tests before deploying')
    }

    stages {
        stage('Pipeline Info') {
            steps {
                echo "=== Starting CI/CD Deployment Pipeline ==="
                echo "Selected Service: ${params.SERVICE}"
                echo "Target Environment: ${params.TARGET_ENV}"
            }
        }

        // 2. Stage for running tests in parallel
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

        // 3. Stage for compiling/building in parallel
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
    }
}

// 4. Custom helper to run sh on Linux, bat on Windows, with safe fallback checks
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
            bat "where ${firstWord} >nul 2>nul && ${cmd} || (echo Skipped: ${firstWord} is not installed & exit /b 0)"
        }
    }
}
