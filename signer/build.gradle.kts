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

// Reverse-engineering tools (not shipped in the image): run with
//   ./gradlew -q research --args="<tool> ..."
val research: SourceSet by sourceSets.creating {
    compileClasspath += sourceSets.main.get().output + sourceSets.main.get().compileClasspath
    runtimeClasspath += output + compileClasspath + sourceSets.main.get().runtimeClasspath
}

tasks.register<JavaExec>("research") {
    classpath = research.runtimeClasspath
    mainClass.set("com.openhongguo.signer.research.Main")
    workingDir = projectDir
    standardInput = System.`in`
}
