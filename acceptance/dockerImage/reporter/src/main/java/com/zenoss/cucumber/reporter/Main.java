package com.zenoss.cucumber.reporter;

import java.io.File;
import java.util.ArrayList;
import java.util.List;
import net.masterthought.cucumber.Configuration;
import net.masterthought.cucumber.ReportBuilder;

/**
 * This is a super simple wrapper around the masterthought's cucumber reporter.
 * The intent is not to replace the Jenkins plugin, but merely to provide the same
 * kind of reports for local builds.
 *
 * For more info, especially for using the Jenkins plugin,
 * @see https://github.com/masterthought/cucumber-reporting
 */
public class Main {
    public static void main(String[] args) throws Exception {

        if (args.length < 2) {
            System.err.println("ERROR: incorrect number of arguments");
            System.err.println("USAGE: reporter outputDirectory jsonReport1 [[jsonReport2]...jsonReportN]");
            System.exit(1);
        }

        File reportOutputDirectory = new File(args[0]);
        List<String> jsonReportFiles = new ArrayList<String>();
        jsonReportFiles.add(args[1]);

        String buildNumber = System.getenv("BUILD_NUMBER");
        String buildProjectName = System.getenv("JOB_NAME");
        Boolean skippedFails = true;   // mark the build failed for skipped tests
        Boolean pendingFails = false;   // don't mark the build failed for pending tests
        Boolean undefinedFails = true;
        Boolean missingFails = false;

        Configuration configuration = new Configuration(reportOutputDirectory, buildProjectName);
        configuration.setStatusFlags(skippedFails, pendingFails, undefinedFails, missingFails);
        configuration.setRunWithJenkins(false);
        configuration.setParallelTesting(false);
        configuration.setBuildNumber(buildNumber);

        ReportBuilder reportBuilder = new ReportBuilder(jsonReportFiles, configuration);
        reportBuilder.generateReports();
    }
}
