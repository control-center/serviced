package com.zenoss.cucumber.reporter;

import java.io.File;
import java.util.ArrayList;
import java.util.List;
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

        String buildNumber = "";
        String buildProjectName = "";
        String pluginUrlPath = "";
        Boolean skippedFails = false;
        Boolean undefinedFails = false;
        Boolean flashCharts = false;
        Boolean runWithJenkins = false;
        Boolean artifactsEnabled = false;
        String artifactConfig = "";
        Boolean highCharts = false;

        ReportBuilder reportBuilder = new ReportBuilder(jsonReportFiles,
            reportOutputDirectory,
            pluginUrlPath,
            buildNumber,
            buildProjectName,
            skippedFails,
            undefinedFails,
            flashCharts,
            runWithJenkins,
            artifactsEnabled,
            artifactConfig,
            highCharts);
        reportBuilder.generateReports();
    }
}
