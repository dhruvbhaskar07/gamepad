Add-Type -AssemblyName System.Drawing

function Make-GamepadIcon($filePath, $size) {
    $bmp = New-Object System.Drawing.Bitmap($size, $size)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $g.Clear([System.Drawing.Color]::Transparent)

    # Background rounded circle
    $rect = New-Object System.Drawing.Rectangle(4, 4, ($size - 8), ($size - 8))
    $bgBrush = New-Object System.Drawing.Drawing2D.LinearGradientBrush($rect, [System.Drawing.Color]::FromArgb(255, 15, 23, 42), [System.Drawing.Color]::FromArgb(255, 6, 8, 14), 45.0)
    $g.FillEllipse($bgBrush, $rect)

    # Outer Neon Cyan Glow Ring
    $penCyan = New-Object System.Drawing.Pen([System.Drawing.Color]::FromArgb(255, 0, 212, 255), [Math]::Max(2, $size / 24))
    $g.DrawEllipse($penCyan, $rect)

    # Controller body outline
    $bodyX = [int]($size * 0.22)
    $bodyY = [int]($size * 0.35)
    $bodyW = [int]($size * 0.56)
    $bodyH = [int]($size * 0.32)
    $bodyRect = New-Object System.Drawing.Rectangle($bodyX, $bodyY, $bodyW, $bodyH)
    $bodyBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 20, 30, 55))
    $g.FillEllipse($bodyBrush, $bodyRect)

    # D-pad cross on left
    $crossPen = New-Object System.Drawing.Pen([System.Drawing.Color]::FromArgb(255, 0, 212, 255), [Math]::Max(2, $size / 20))
    $cx = [int]($size * 0.35)
    $cy = [int]($size * 0.51)
    $arm = [int]($size * 0.07)
    $g.DrawLine($crossPen, ($cx - $arm), $cy, ($cx + $arm), $cy)
    $g.DrawLine($crossPen, $cx, ($cy - $arm), $cx, ($cy + $arm))

    # Action Buttons on right
    $bx = [int]($size * 0.65)
    $by = [int]($size * 0.51)
    $bRad = [int]($size * 0.035)
    $btnCyan = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 0, 220, 255))
    $btnRed = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 255, 0, 85))

    $g.FillEllipse($btnCyan, ($bx - $bRad), ($by + $arm - $bRad), ($bRad * 2), ($bRad * 2))
    $g.FillEllipse($btnRed, ($bx + $arm - $bRad), ($by - $bRad), ($bRad * 2), ($bRad * 2))

    $dir = Split-Path -Parent $filePath
    if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
    $bmp.Save($filePath, [System.Drawing.Imaging.ImageFormat]::Png)
    $g.Dispose()
    $bmp.Dispose()
}

Make-GamepadIcon "d:\vib\android\src\main\res\mipmap-xxhdpi\ic_launcher.png" 192
Make-GamepadIcon "d:\vib\android\src\main\res\mipmap-xhdpi\ic_launcher.png" 96
Make-GamepadIcon "d:\vib\android\src\main\res\mipmap-hdpi\ic_launcher.png" 72
Make-GamepadIcon "d:\vib\android\src\main\res\mipmap-mdpi\ic_launcher.png" 48
Write-Host "Launcher icons generated successfully!"
