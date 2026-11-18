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
    System.getProperty("preview")?.let { systemProperty("preview", it) }
    System.getProperty("traceFirst")?.let { systemProperty("traceFirst", it) }
    System.getProperty("dump")?.let { systemProperty("dump", it) }
    System.getProperty("from")?.let { systemProperty("from", it) }
    System.getProperty("url")?.let { systemProperty("url", it) }
    System.getProperty("secs")?.let { systemProperty("secs", it) }
    System.getProperty("viaLdrb")?.let { systemProperty("viaLdrb", it) }
    System.getProperty("earlyHook")?.let { systemProperty("earlyHook", it) }
    System.getProperty("regfile")?.let { systemProperty("regfile", it) }
    System.getProperty("memprobe")?.let { systemProperty("memprobe", it) }
    System.getProperty("flips")?.let { systemProperty("flips", it) }
    System.getProperty("watchaddr")?.let { systemProperty("watchaddr", it) }
    System.getProperty("variant")?.let { systemProperty("variant", it) }
    System.getProperty("to")?.let { systemProperty("to", it) }
}

tasks.register("researchClasspath") {
    dependsOn("researchClasses")
    doLast { println(research.runtimeClasspath.asPath) }
}
