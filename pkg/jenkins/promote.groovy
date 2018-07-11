#!groovy
//
// promote.groovy - used in Jenkins to promote a serviced build from one maturity to the next.
//
pipeline {

    agent {
        dockerfile {
            dir 'pkg/reversion/deb'
        }
    }

    parameters {
        choice(name: 'SOURCE_MATURITY', choices: 'unstable\ntesting', description: 'The maturity for packages to be promoted.')
        string(name: 'SOURCE_VERSION', description: 'If looking for unstable, provide the build number of the desired \'merge-start\' serviced build.')
        choice(name: 'TARGET_MATURITY', choices: 'testing\nstable', description: 'The maturity for promoted packages.')
        string(name: 'TARGET_VERSION', description: '')
    }

    stages {
        stage('Fetch source artifact') {
            steps {
                googleStorageDownload(credentialsId: 'zing-registry-188222', bucketUri: 'gs://cz-${params.SOURCE_MATURITY}/serviced/${params.SOURCE_VERSION}/*.deb', localDirectory: '.')
            }
        }

        stage('Repackage') {
            steps {
                sh """
                    rm -rf output
                    ls -la
                    dpkg-deb -f *.deb
                    FILE=`ls *.deb`
                    mkdir -p output && cd output
                    echo -e "\\nMetadata for the source package"
                    dpkg -f "${FILE}"
                    
                    # run reversion
                    dpkg-reversion -b -v ${params.TARGET_VERSION} "${FILE}"

                    echo -e "\\nMetadata for the source package"
                    serviced_${params.TARGET_VERSION}_amd64.deb 
                """
            }
        }

        stage('Promote artifact') {
            steps {
                googleStorageUpload(credentialsId: 'zing-registry-188222', bucket: 'gs://cz-${params.TARGET_MATURITY}/serviced', pattern:'output/*.deb', pathPrefix: 'output')
            }
        }
    }

}