plugins {
    java
    application
}

repositories {
    mavenCentral()
    maven("https://jitpack.io")
}

dependencies {
    implementation("com.github.zhkl0228.unidbg:unidbg-android:0.9.8")
}

application {
    mainClass.set("com.openhongguo.signer.SignerServer")
}

java {
    toolchain {
        languageVersion.set(JavaLanguageVersion.of(21))
    }
}

tasks.withType<JavaCompile> {
    options.release.set(21)
}
