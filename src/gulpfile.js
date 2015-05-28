var gulp   = require("gulp");
var mainBowerFiles = require('main-bower-files');
var gulpgo = require("gulp-go");
var livereload = require('gulp-livereload');
var less = require('gulp-less');
var path = require('path');
var uglify = require('gulp-uglify');

var go;

gulp.task("go-run", function() {
  go = gulpgo.run("main.go", [], {cwd: __dirname, stdio: 'inherit', godep: true});
});

gulp.task("bower-files", function(){
    gulp.src(mainBowerFiles()).pipe(gulp.dest("./public/lib"));
});

gulp.task('less', function() {
  gulp.src('./less/**/*.less')
    .pipe(less({
      paths: [ path.join(__dirname, 'less', 'includes') ]
    }))
    .pipe(gulp.dest('./public/css'))
    .pipe(livereload());
});

gulp.task('compress', function() {
  return gulp.src('scripts/*.js')
    .pipe(uglify())
    .pipe(gulp.dest('public/scripts'))
    .pipe(livereload());
});

gulp.task("watch", function() {
  livereload.listen();
  gulp.watch('./less/**/*.less', ['less']);
  gulp.watch('./scripts/**/*.js', ['compress']);
  gulp.watch([__dirname+"/**/*.go"]).on("change", function() {
    go.restart();
  });
})

gulp.task("default", ["bower-files", "go-run", "watch", ]);
