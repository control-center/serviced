module.exports = function(grunt){
    grunt.initConfig({
        pkg: grunt.file.readJSON('package.json'),
        concat: {
            js: {
                src: ['static/js/main.js', 'static/js/controllers/*.js'],
                dest: 'static/js/controlplane.js'
            }
        },
        uglify: {
            options: {
                mangle: false,
                banner: '/*! <%= pkg.name %> <%= grunt.template.today("yyyy-mm-dd") %> */\n'
            },
            build: {
                src: 'static/js/controlplane.js',
                dest: 'static/js/controlplane.min.js'
            }
        },
        watch: {
            dev: {
                options: {
                    livereload: true
                },
                files: ["**/*", "!static/js/controlplane.js", "!static/js/controlplane.min.js"],
                tasks: ["concat"]
            }
        }
    });

    grunt.loadNpmTasks('grunt-contrib-concat');
    grunt.loadNpmTasks('grunt-contrib-uglify');
    grunt.loadNpmTasks('grunt-contrib-watch');

    grunt.registerTask('default', ['concat', 'uglify']);
}