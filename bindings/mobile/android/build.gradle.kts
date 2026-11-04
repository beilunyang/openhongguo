// openhongguo 的 Android 库。先运行 ../build.sh android，把 Go 编译出的
// core 库放进 libs/ 和 src/main/jniLibs/，再用 Gradle 构建。
plugins {
    id("com.android.library") version "8.13.2"
    id("org.jetbrains.kotlin.android") version "2.1.20"
    id("com.vanniktech.maven.publish") version "0.37.0"
}

android {
    namespace = "dev.openhongguo"
    compileSdk = 36

    defaultConfig {
        minSdk = 21
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        consumerProguardFiles("consumer-rules.pro")
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    testOptions {
        unitTests.isReturnDefaultValues = true
    }
}

kotlin {
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
    }
}

dependencies {
    // gomobile 生成的 Java 类（dev.openhongguo.core 和 go.*），随 aar 一起打包。
    implementation(files("libs/core.jar"))
    api("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.10.2")

    testImplementation("junit:junit:4.13.2")
    testImplementation("org.json:json:20250517")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.10.2")

    androidTestImplementation("androidx.test:runner:1.6.2")
    androidTestImplementation("androidx.test.ext:junit:1.2.1")
}

// 发布到 Maven Central：./gradlew publishToMavenCentral。凭据和签名密钥放在
// ~/.gradle/gradle.properties：mavenCentralUsername、mavenCentralPassword、
// signingInMemoryKey、signingInMemoryKeyPassword。没有签名密钥时不签名，
// 只能发布到本地（./gradlew publishToMavenLocal）。
mavenPublishing {
    coordinates("io.github.beilunyang", "openhongguo", "0.2.0")
    publishToMavenCentral()
    if (providers.gradleProperty("signingInMemoryKey").isPresent) {
        signAllPublications()
    }
    pom {
        name.set("openhongguo")
        description.set("红果短剧（Hongguo）的非官方 Android SDK：匿名浏览、搜索与播放，基于 Go SDK 编译的原生库")
        inceptionYear.set("2026")
        url.set("https://github.com/beilunyang/openhongguo")
        licenses {
            license {
                name.set("Apache License 2.0")
                url.set("https://www.apache.org/licenses/LICENSE-2.0.txt")
            }
        }
        developers {
            developer {
                id.set("beilunyang")
                name.set("beilunyang")
                url.set("https://github.com/beilunyang")
            }
        }
        scm {
            url.set("https://github.com/beilunyang/openhongguo")
            connection.set("scm:git:https://github.com/beilunyang/openhongguo.git")
            developerConnection.set("scm:git:ssh://git@github.com/beilunyang/openhongguo.git")
        }
    }
}
