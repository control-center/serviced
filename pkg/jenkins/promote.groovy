#!groovy
//
// promote.groovy - used in Jenkins to promote a serviced build from one maturity to the next.
//
pipeline {

    agent {
        label 'docker-centos-7-4'
    }

    parameters {
        choice(name: 'SOURCE_MATURITY', choices: 'unstable\ntesting', description: 'The maturity for the package to be promoted.')
        string(name: 'SOURCE_VERSION', description: 'If looking for unstable, provide the build number from the \'merge-start\' \
                serviced build for the artifact. For testing, provide just the version part of the deb filename: \
                The <b>1.6.0-RC0</b> from serviced_1.6.0-RC0_amd64.deb')
        choice(name: 'TARGET_MATURITY', choices: 'testing\nstable', description: 'The maturity for the promoted package.')
        string(name: 'TARGET_VERSION', description: 'e.g. 1.6.0')
    }

    stages {
        stage('Fetch source artifact') {
            steps {
                echo "SOURCE_MATURITY = ${params.SOURCE_MATURITY}"
                echo "SOURCE_VERSION = ${params.SOURCE_VERSION}"
                echo "TARGET_MATURITY = ${params.TARGET_MATURITY}"
                echo "TARGET_VERSION = ${params.TARGET_VERSION}"

                script {
                    path = "serviced_${params.SOURCE_VERSION}_amd64"
                    pathPrefix = "serviced"
                    if ("${params.SOURCE_MATURITY}" == 'unstable') {
                        path = "${params.SOURCE_VERSION}/*"
                        pathPrefix = "serviced/${params.SOURCE_VERSION}"
                    }
                    uri = "gs://cz-${params.SOURCE_MATURITY}/serviced/${path}.deb"
                }

                echo "URL is ${uri}"

                googleStorageDownload(credentialsId: 'zing-registry-188222', bucketUri: "${uri}", localDirectory: '.', pathPrefix: "${pathPrefix}")
                script {
                    debfile = sh returnStdout: true, script: "ls *.deb"
                }
            }
        }
        stage('Repackage and upload') {
            steps {
                script {
                    try  {
                        sh """
                            docker build -t zenoss/serviced-promote:deb $WORKSPACE/pkg/reversion/deb
            
                            sudo mkdir -p input
                            sudo mv *.deb input
                            sudo mkdir -p output
                            
                            docker run -v $WORKSPACE/output:/output -v $WORKSPACE/input:/input zenoss/serviced-promote:deb \
                                bash -c "cd /output && deb-reversion -b -v ${params.TARGET_VERSION}-${BUILD_NUMBER} /input/$debfile"
                        """
                        uri = "gs://cz-${params.TARGET_MATURITY}/serviced/"
                        googleStorageUpload(credentialsId: 'zing-registry-188222', bucket: "${uri}", pattern:'output/*.deb', pathPrefix: 'output')
                    }
                    finally {
                        sh """
                            sudo rm -rf input
                            sudo rm -rf output
                        """
                    }
                }
            }
        }
    }
}
