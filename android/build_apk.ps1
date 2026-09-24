$ErrorActionPreference = "Stop"

$sdkRoot = "C:\Users\WINTER\AppData\Local\Android\Sdk"
$buildToolsVersion = "35.0.0"
$platformVersion = "android-34"

$aapt2 = "$sdkRoot\build-tools\$buildToolsVersion\aapt2.exe"
$d8 = "$sdkRoot\build-tools\$buildToolsVersion\d8.bat"
$zipalign = "$sdkRoot\build-tools\$buildToolsVersion\zipalign.exe"
$apksigner = "$sdkRoot\build-tools\$buildToolsVersion\apksigner.bat"
$androidJar = "$sdkRoot\platforms\$platformVersion\android.jar"
$keystore = "C:\Users\WINTER\.android\debug.keystore"

$projectRoot = "d:\vib"
$androidDir = "$projectRoot\android"
$buildDir = "$androidDir\build"
$outputApk = "$projectRoot\DualSenseMobile.apk"

Write-Host "================================================" -ForegroundColor Cyan
Write-Host " Building DualSense Mobile Android APK...       " -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan

# 0. Clean & setup build dirs
if (Test-Path $buildDir) { Remove-Item -Recurse -Force $buildDir }
New-Item -ItemType Directory -Force -Path "$buildDir\gen", "$buildDir\classes", "$buildDir\dex" | Out-Null

# 1. Compile Resources with aapt2
Write-Host "[1/6] Compiling Android Resources (AAPT2)..." -ForegroundColor Yellow
& $aapt2 compile --dir "$androidDir\src\main\res" -o "$buildDir\res.zip"
if ($LASTEXITCODE -ne 0) { throw "AAPT2 compile failed" }

# 2. Link Resources & Generate R.java
Write-Host "[2/6] Linking Resources & Packaging Manifest..." -ForegroundColor Yellow
& $aapt2 link -I $androidJar `
    "$buildDir\res.zip" `
    --manifest "$androidDir\src\main\AndroidManifest.xml" `
    -A "$androidDir\src\main\assets" `
    --java "$buildDir\gen" `
    --min-sdk-version 21 `
    --target-sdk-version 34 `
    --version-code 1 `
    --version-name "1.0.0" `
    -o "$buildDir\app-unsigned-unaligned.apk"
if ($LASTEXITCODE -ne 0) { throw "AAPT2 link failed" }

# 3. Compile Java Source Files
Write-Host "[3/6] Compiling Java Sources (javac)..." -ForegroundColor Yellow
$javaSources = @(
    "$buildDir\gen\com\dualsense\gamepad\R.java",
    "$androidDir\src\main\java\com\dualsense\gamepad\MainActivity.java"
)
& javac -encoding UTF-8 -source 17 -target 17 `
    -cp $androidJar `
    -d "$buildDir\classes" `
    $javaSources
if ($LASTEXITCODE -ne 0) { throw "Java compilation failed" }

# 4. Dex Classes with D8
Write-Host "[4/6] Converting Java Bytecode to Dalvik Executable (D8)..." -ForegroundColor Yellow
$classFiles = Get-ChildItem "$buildDir\classes" -Recurse -Filter *.class | Select-Object -ExpandProperty FullName
& $d8 --min-api 21 --lib $androidJar --output "$buildDir\dex" $classFiles
if ($LASTEXITCODE -ne 0) { throw "D8 dexing failed" }

# 5. Insert classes.dex into APK
Write-Host "[5/6] Injecting classes.dex into APK package..." -ForegroundColor Yellow
Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem

$zip = [System.IO.Compression.ZipFile]::Open("$buildDir\app-unsigned-unaligned.apk", [System.IO.Compression.ZipArchiveMode]::Update)
[System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile($zip, "$buildDir\dex\classes.dex", "classes.dex")
$zip.Dispose()

# 6. Zipalign & Sign
Write-Host "[6/6] Zipaligning & Signing with APKSigner..." -ForegroundColor Yellow
& $zipalign -f -p 4 "$buildDir\app-unsigned-unaligned.apk" "$buildDir\app-aligned.apk"
if ($LASTEXITCODE -ne 0) { throw "zipalign failed" }

if (-not (Test-Path $keystore)) {
    Write-Host "Creating debug keystore..." -ForegroundColor Cyan
    & keytool -genkey -v -keystore $keystore -storepass android -alias androiddebugkey -keypass android -keyalg RSA -keysize 2048 -validity 10000 -dname "CN=Android Debug,O=Android,C=US"
}

& $apksigner sign `
    --ks $keystore `
    --ks-pass pass:android `
    --key-pass pass:android `
    --out $outputApk `
    "$buildDir\app-aligned.apk"
if ($LASTEXITCODE -ne 0) { throw "APK signing failed" }

$apkItem = Get-Item $outputApk
$apkSizeMB = [math]::Round($apkItem.Length / 1MB, 2)

Write-Host "================================================" -ForegroundColor Green
Write-Host " SUCCESS! DualSense Mobile APK Created!         " -ForegroundColor Green
Write-Host " Location: $outputApk                           " -ForegroundColor Green
Write-Host " Size: $apkSizeMB MB ($($apkItem.Length) bytes) " -ForegroundColor Green
Write-Host "================================================" -ForegroundColor Green
