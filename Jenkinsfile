pipeline {
    // 1. 'agent' tells Jenkins where to run the job (any available machine or container)
    agent any

    // 2. 'stages' is a container for all individual build phases
    stages {
        
        // 3. 'stage' represents a single phase of execution (e.g. Info, Test, Deploy)
        stage('Environment Info') {
            
            // 4. 'steps' defines the list of commands to run inside this stage
            steps {
                echo 'Checking local system environment...'
                
                // 5. 'script' allows running dynamic Groovy logic (like if/else)
                script {
                    if (isUnix()) {
                        sh 'uname -a || true'
                    } else {
                        bat 'ver'
                    }
                }
            }
        }
    }
}
