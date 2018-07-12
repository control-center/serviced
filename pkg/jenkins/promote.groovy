#!groovy
//
// promote.groovy - used in Jenkins to promote a serviced build from one maturity to the next.
//
pipeline {

    agent {
        label 'docker-centos-7-4'
    }

    parameters {
        choice(name: 'SOURCE_MATURITY', choices: 'unstable\ntesting', description: 'The maturity for packages to be promoted.')
        string(name: 'SOURCE_VERSION', description: 'If looking for unstable, provide the build number of the desired \'merge-start\' serviced build.')
        choice(name: 'TARGET_MATURITY', choices: 'testing\nstable', description: 'The maturity for promoted packages.')
        string(name: 'TARGET_VERSION', description: 'e.g. 1.6.0')
        string(name: 'RELEASE_PHASE', description: 'RC1, RC2, etc.')
    }

    stages {
        stage('Fetch source artifact') {
            steps {
                echo "SOURCE_MATURITY = ${params.SOURCE_MATURITY}"
                echo "SOURCE_VERSION = ${params.SOURCE_VERSION}"
                echo "TARGET_MATURITY = ${params.TARGET_MATURITY}"
                echo "TARGET_MATURITY = ${params.TARGET_MATURITY}"

                script {
                    path = ${params.SOURCE_VERSION}
                    if (${params.SOURCE_MATURITY} == 'unstable') {
                        path = ${params.SOURCE_VERSION} + "/*"
                    }
                    uri = "gs://cz-${params.SOURCE_MATURITY}/serviced/${path}.deb"
                }

                echo "URL is $uri"

                googleStorageDownload(credentialsId: 'zing-registry-188222', bucketUri: $uri, localDirectory: '.')
            }
        }

        stage('Repackage') {
            steps {
                sh """
                    docker build -t zenoss/serviced-promote:deb $WORKSPACE/pkg/reversion/deb
    
                    sudo mkdir -p input
                    sudo mv *.deb input
                    sudo mkdir -p output
                    
                    docker run -v $WORKSPACE/output:/output -v $WORKSPACE/input:/input zenoss/serviced-promote:deb \
                        bash -c "cd /output && deb-reversion -b -v ${params.TARGET_VERSION}-${param.RELEASE_PHASE} /input/$debfile"
                """
            }
        }

        stage('Promote artifact') {
            steps {
                script {
                    uri = "gs://cz-${params.TARGET_MATURITY}/serviced/"
                }

                googleStorageUpload(credentialsId: 'zing-registry-188222', bucket: "${uri}", pattern:'output/*.deb', pathPrefix: 'output')
            }
        }
    }
    post {
        cleanup {
            sh """
                sudo rm -rf output
                sudo rm -rf input
            """
        }
    }

}