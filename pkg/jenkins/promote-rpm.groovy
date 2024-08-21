#!groovy
//
// promote-rpm.groovy - used in Jenkins to promote a serviced build from one maturity to the next.
//
pipeline {

    agent {
        label 'docker-centos-7-4'
    }

    parameters {
        choice(name: 'SOURCE_MATURITY', choices: 'unstable\ntesting', description: 'The maturity for the package to be promoted.')
        string(name: 'SOURCE_VERSION', description: 'If looking for unstable, provide the version part and the build number from the \'merge-start\' \
                serviced build for the artifact: \
                The <b>1.10.5-0.0.37</b> from serviced-1.10.5-0.0.37.unstable.x86_64.rpm\
                For testing, provide just the version part of the rpm filename: \
                The <b>1.10.5-0.0.RC1</b> from serviced-1.10.5-0.0.RC1.x86_64.rpm')
        choice(name: 'TARGET_MATURITY', choices: 'testing\nstable', description: 'The maturity for the promoted package.')
        string(name: 'TARGET_VERSION', description: 'e.g. 1.10.5-1 or 1.10.5-0.0.RC1')
    }

    stages {
        stage('Fetch source artifact') {
            steps {
                echo "SOURCE_MATURITY = ${params.SOURCE_MATURITY}" //unstable
                echo "SOURCE_VERSION = ${params.SOURCE_VERSION}"   //1.10.5-0.0.37
                echo "TARGET_MATURITY = ${params.TARGET_MATURITY}" //testing
                echo "TARGET_VERSION = ${params.TARGET_VERSION}"   //1.10.5-0.0.RC1

                script {
                    path = "serviced-${params.SOURCE_VERSION}.${params.SOURCE_MATURITY}.x86_64"
                    if ("${params.SOURCE_MATURITY}" == 'testing') {
                        path = "serviced-${params.SOURCE_VERSION}.x86_64"
                    }
                    pathPrefix = "yum/zenoss/${params.SOURCE_MATURITY}/centos/el7/os/x86_64/"
                    uri = "gs://get-zenoss-io/yum/zenoss/${params.SOURCE_MATURITY}/centos/el7/os/x86_64/${path}.rpm"
                }

                echo "URL is ${uri}"

                googleStorageDownload(credentialsId: 'zing-registry-188222', bucketUri: "${uri}", localDirectory: '.', pathPrefix: pathPrefix)
                script {
                    rpmfile = sh returnStdout: true, script: "ls *.rpm"
                }
            }
        }
        stage('Repackage and upload') {
            steps {
                script {
                    (TARGET_VERSION, TARGET_RELEASE) = params.TARGET_VERSION.tokenize("-")
                    ESCAPED_VERSION = TARGET_VERSION.replaceAll(/\./, /\\\./)
                    VERSION_SED_CMD = "s/^Version:.*/Version:${ESCAPED_VERSION}/"
                    try  {
                        sh """
                            docker build -t zenoss/serviced-promote:rpm $WORKSPACE/pkg/reversion/rpm
            
                            sudo mkdir -p input
                            sudo mv *.rpm input
                            sudo mkdir -p output
                            
                            echo -E "${VERSION_SED_CMD}" > sed.cmd
                            sudo mv sed.cmd input
                            cat input/sed.cmd
                   
                            docker run -v $WORKSPACE/output:/output -v $WORKSPACE/input:/input zenoss/serviced-promote:rpm \
                                bash -c "cd /output && rpmrebuild --release="${TARGET_RELEASE}" --notest-install --directory=/output --change-spec-preamble='sed -f /input/sed.cmd' --package /input/${rpmfile}"
                        """
                        uri = "gs://get-zenoss-io/yum/zenoss/${params.TARGET_MATURITY}/centos/el7/os/x86_64/"
                        echo "URL is ${uri}"
                        sh "ls output/x86_64/*.rpm"
                        googleStorageUpload(credentialsId: 'zing-registry-188222', bucket: "${uri}", pattern:'output/x86_64/*.rpm', pathPrefix: 'output/x86_64/')
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
