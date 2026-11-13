Pod::Spec.new do |s|
  s.name             = 'openhongguo'
  s.version          = '0.3.0'
  s.summary          = '红果短剧（Hongguo）的非官方 Flutter SDK'
  s.description      = '匿名浏览、搜索与播放，基于 gomobile 编译的 Go SDK。'
  s.homepage         = 'https://github.com/beilunyang/openhongguo'
  s.license          = { :type => 'Apache-2.0' }
  s.author           = 'beilunyang'
  s.source           = { :path => '.' }
  s.source_files     = 'Classes/**/*'
  s.dependency 'Flutter'
  s.platform = :ios, '15.0'
  # gomobile 编译的 Go core，由 bindings/mobile/build.sh flutter 放进来。
  s.vendored_frameworks = 'OpenhongguoCore.xcframework'
  s.static_framework = true

  # Flutter.framework does not contain a i386 slice.
  s.pod_target_xcconfig = { 'DEFINES_MODULE' => 'YES', 'EXCLUDED_ARCHS[sdk=iphonesimulator*]' => 'i386' }
  s.swift_version = '5.9'
end
