/*
  Copyright 2016 The LUCI Authors. All rights reserved.
  Use of this source code is governed under the Apache License, Version 2.0
  that can be found in the LICENSE file.

  This document has been largely derived from the Polymer Starter Kit:
  https://github.com/PolymerElements/polymer-starter-kit
*/

'use strict';

var path = require('path');
var argv = require('yargs').argv;

var exports = module.exports = {}
exports.base = path.join(__dirname, '..');
exports.out = (argv.out || exports.base);
exports.plugins = require('gulp-load-plugins')({
  config: path.join(exports.base, 'package.json'),
});

// Include Gulp & tools we'll use
var $ = exports.plugins;
var browserSync = require('browser-sync');
var cleanCSS = require('gulp-clean-css');
var debug = require('gulp-debug');
var del = require('del');
var format = require('gulp-clang-format');
var fs = require('fs');
var glob = require('glob-all');
var gulpIf = require('gulp-if');
var htmlmin = require('gulp-htmlmin');
var historyApiFallback = require('connect-history-api-fallback');
var hyd = require('hydrolysis');
var merge = require('merge-stream');
var reload = browserSync.reload;
var ts = require('gulp-typescript');
var sourcemaps = require('gulp-sourcemaps');
var tslint = require('gulp-tslint');


var AUTOPREFIXER_BROWSERS = [
  'ie >= 10',
  'ie_mob >= 10',
  'ff >= 30',
  'chrome >= 34',
  'safari >= 7',
  'opera >= 23',
  'ios >= 7',
  'android >= 4.4',
  'bb >= 10'
];

// Include directory (/web/inc/)
exports.incDir = path.join(exports.base, 'inc');

// Common (global) tasks.
exports.setup_common = function(gulp) {
  // Verify TypeScript file integrity and formatting.
  gulp.task('tslint', function() {
    process.chdir(exports.incDir);

    return gulp.src(['**/*.ts', '!bower_components/**'])
        .pipe(tslint({
            configuration: './tslint.json',
            formatter: 'verbose',
        }))
        .pipe(tslint.report());
  });

  gulp.task('check-ts', function() {
    process.chdir(exports.incDir);

    // Transpile each TypeScript module independently into JavaScript.
    var tsconfigPath = path.join(exports.incDir, 'tsconfig.json');

    // Compile the files in "scripts-ts/*.ts" into a single out file.
    var tsProj = ts.createProject(tsconfigPath, {
      typeRoots: [path.join(exports.base, 'node_modules', '@types')],
    });

    return gulp.src(['**/*.ts', '!bower_components/**'])
        .pipe(tsProj());
  });

  gulp.task('check-format', function() {
    process.chdir(exports.incDir);
    return gulp.src(['**/*.ts', '!bower_components/**'])
        .pipe(format.checkFormat())
        .on('warning', function(e) {
          process.stdout.write(e.message);
          process.exit(1)
        });
  });

  gulp.task('format', function() {
    process.chdir(exports.incDir);

    // The base option ensures the glob doesn't strip prefixes.
    return gulp.src(['**/*.ts', '!bower_components/**'])
        .pipe(format.format())
        .pipe(gulp.dest('.'));
  });

  // Build production files, the default task
  gulp.task('lint', gulp.series(['tslint']));

  // Build production files, the default task
  gulp.task('presubmit', gulp.parallel(['lint', 'check-format', 'check-ts']));
};

// Project-specific tasks.
exports.setup = function(gulp, appDir, config) {
  var APP = path.basename(appDir);
  var DIST = path.join(exports.out, 'dist', APP);

  var layout = {
    app: APP,
    dir: process.cwd(),
    web: '../..',
    inc: './inc',
    distPath: DIST,

    // NOTE: Takes vararg via "arguments".
    dist: function() {
      return extendPath(DIST).apply(null, arguments);
    },
  };

  var extendPath = function() {
    var base = [].slice.call(arguments);
    return function() {
      // Simple case: only base, no additional elements.
      if (base.length === 1 && arguments.length === 0) {
        return base[0];
      }

      var parts = base.concat();
      parts.push.apply(parts, arguments)
      return path.join.apply(null, parts);
    };
  };

  var styleTask = function(stylesPath, srcs) {
    return gulp.src(srcs.map(function(src) {
        return path.join(stylesPath, src);
      }))
      .pipe($.autoprefixer(AUTOPREFIXER_BROWSERS))
      .pipe(gulp.dest('.tmp/' + stylesPath))
      .pipe(cleanCSS())
      .pipe(gulp.dest(layout.dist(stylesPath)))
      .pipe($.size({title: stylesPath}));
  };

  var imageOptimizeTask = function(src, dest) {
    return gulp.src(src)
      .pipe($.imagemin({
        progressive: true,
        interlaced: true
      }))
      .pipe(gulp.dest(dest))
      .pipe($.size({title: 'images'}));
  };

  var optimizeHtmlTask = function(src, dest) {
    var assets = $.useref.assets({
      searchPath: ['.tmp', '.']
    });

    return gulp.src(src)
      .pipe(assets)
      // Concatenate and minify JavaScript
      .pipe($.if('*.js', $.uglify({
        preserveComments: 'some'
      })))
      // Concatenate and minify styles
      // In case you are still using useref build blocks
      .pipe($.if('*.css', cleanCSS()))
      .pipe(assets.restore())
      .pipe($.useref())
      // Minify any HTML
      .pipe($.if('*.html', htmlmin({
        remoteAttributeQuotes: false,
        remoteEmptyAttributes: false,
        remoteRedundantAttributes: false,
      })))
      // Output files
      .pipe(gulp.dest(dest))
      .pipe($.size({
        title: 'html'
      }));
  };

  // Transpiles "inc/*/*.ts" and deposits the result alongside their source
  // "ts" files.
  gulp.task('ts', function() {
    // Compile the files in "scripts-ts/*.ts" into a single out file.
    var appTsDir = path.join(layout.inc, 'apps', layout.app);
    var tsProj = ts.createProject(path.join(layout.inc, 'tsconfig.json'), {
      typeRoots: [path.join(exports.base, 'node_modules', '@types')],
      outFile: path.join(appTsDir, 'main.js'),
    });

    return gulp.src(path.join(layout.inc, 'apps', layout.app, '*.ts'), {
          exclude: ['*_test.ts'],
        })
        .pipe(sourcemaps.init())
        .pipe(tsProj())
        .pipe(sourcemaps.write('.'))
        .pipe(gulp.dest(layout.inc))
  });

  // Compile and automatically prefix stylesheets
  gulp.task('styles', function() {
    return styleTask('styles', ['**/*.css']);
  });

  gulp.task('elements', function() {
    return styleTask('elements', ['**/*.css']);
  });

  // Optimize images
  gulp.task('images', function() {
    return imageOptimizeTask('images/**/*', layout.dist('images'));
  });

  // Copy all files at the root level (app)
  gulp.task('copy', function() {
    // Application files.
    var app = gulp.src([
      '*',
      '!inc',
      '!test',
      '!elements',
      '!inc/bower_components',
      '!cache-config.json',
      '!**/.DS_Store',
      '!gulpfile.js',
      '!package.json',
    ]).pipe(gulp.dest(layout.dist()));

    // Copy over only the bower_components we need
    // These are things which cannot be vulcanized
    var webcomponentsjs = gulp.src([
      'inc/bower_components/webcomponentsjs/webcomponents-lite.min.js',
    ]).pipe(gulp.dest(layout.dist('inc/bower_components/webcomponentsjs/')));

    var requirejs = gulp.src([
      'inc/bower_components/requirejs/require.js',
    ]).pipe(gulp.dest(layout.dist('inc/bower_components/requirejs/')));

    var includes = (config.includes) ? (config.includes(gulp, layout)) : ([]);
    return merge(app, includes, webcomponentsjs, requirejs)
      .pipe($.size({
        title: 'copy'
      }));
  });

  // Copy web fonts to dist
  gulp.task('fonts', function() {
    return gulp.src(['fonts/**'])
      .pipe(gulp.dest(layout.dist('fonts')))
      .pipe($.size({
        title: 'fonts'
      }));
  });

  // Scan your HTML for assets & optimize them
  gulp.task('html', function() {
    return optimizeHtmlTask(
      ['**/*.html', '!{elements,test,inc}/**/*.html'],
      layout.dist());
  });

  gulp.task('compile', gulp.series(['ts']));

  // Vulcanize granular configuration
  gulp.task('vulcanize', gulp.series(['compile'], function vulcanize() {
    var fsResolver = hyd.FSResolver
    return gulp.src('elements/elements.html')
      .pipe($.vulcanize({
        stripComments: true,
        inlineCss: true,
        inlineScripts: true,
      }))
      .pipe(gulp.dest(layout.dist('elements')))
      .pipe($.size({title: 'vulcanize'}));
  }));

  // Clean output directory
  gulp.task('clean', function() {
    var dist = layout.dist();
    var remove = ['.tmp', path.join(dist, '*')];
    var keep = '!'+path.join(dist, '.keep');
    return del(remove.concat(keep), {force: true, dot:true});
  });

  // Build production files, the default task
  gulp.task('default', gulp.series([
    'clean',
    gulp.parallel(['copy', 'styles', 'images', 'fonts', 'html', 'compile', 'vulcanize']),
  ]));


  // Watch files for changes & reload
  gulp.task('serve', gulp.series(['default'], function() {
    browserSync({
      port: 5000,
      ui: {
        port: 5080,
      },
      notify: false,
      logPrefix: 'PSK',
      snippetOptions: {
        rule: {
          match: '<span id="browser-sync-binding"></span>',
          fn: function(snippet) {
            return snippet;
          }
        }
      },
      // Run as an https by uncommenting 'https: true'
      // Note: this uses an unsigned certificate which on first access
      //       will present a certificate warning in the browser.
      // https: true,
      server: {
        baseDir: ['.'],
        middleware: [historyApiFallback()]
      }
    });

    gulp.watch(['**/*.html'], ['html', reload]);
    gulp.watch(['styles/**/*.css'], ['styles', reload]);
    gulp.watch(['elements/**/*.css'], ['elements', reload]);
    gulp.watch(['images/**/*'], [reload]);
    gulp.watch(['inc/**/*.ts'], { cwd: exports.base }, ['compile', reload]);
  }));

  // Build and serve the output from the dist build
  gulp.task('serve:dist', gulp.series(['default'], function() {
    browserSync({
      port: 5000,
      ui: {
        port: 5080,
      },
      notify: false,
      logPrefix: 'PSK',
      snippetOptions: {
        rule: {
          match: '<span id="browser-sync-binding"></span>',
          fn: function(snippet) {
            return snippet;
          }
        }
      },
      // Run as an https by uncommenting 'https: true'
      // Note: this uses an unsigned certificate which on first access
      //       will present a certificate warning in the browser.
      // https: true,
      server: layout.dist(),
      middleware: [historyApiFallback()]
    });
  }));


  // Install common targets.
  exports.setup_common(gulp);
};

// Load custom tasks from the `tasks` directory
try {
  require('require-dir')('tasks');
} catch (err) {}
