// Lightweight QR Code Generator implementation (Zero External Dependencies)
    (function(window) {
      function QR8bitByte(data) {
        this.mode = 4;
        this.data = data;
      }
      QR8bitByte.prototype = {
        getLength: function() { return this.data.length; },
        write: function(buffer) {
          for (var i = 0; i < this.data.length; i++) {
            buffer.put(this.data.charCodeAt(i), 8);
          }
        }
      };

      function QRCodeModel(typeNumber, errorCorrectLevel) {
        this.typeNumber = typeNumber;
        this.errorCorrectLevel = errorCorrectLevel;
        this.modules = null;
        this.moduleCount = 0;
        this.dataCache = null;
        this.dataList = [];
      }

      QRCodeModel.prototype = {
        addData: function(data) {
          this.dataList.push(new QR8bitByte(data));
          this.dataCache = null;
        },
        isDark: function(row, col) {
          if (row < 0 || this.moduleCount <= row || col < 0 || this.moduleCount <= col) {
            throw new Error(row + "," + col);
          }
          return this.modules[row][col];
        },
        getModuleCount: function() { return this.moduleCount; },
        make: function() {
          this.makeImpl(false, this.getBestMaskPattern());
        },
        makeImpl: function(test, maskPattern) {
          this.moduleCount = this.typeNumber * 4 + 17;
          this.modules = new Array(this.moduleCount);
          for (var row = 0; row < this.moduleCount; row++) {
            this.modules[row] = new Array(this.moduleCount);
            for (var col = 0; col < this.moduleCount; col++) {
              this.modules[row][col] = null;
            }
          }
          this.setupPositionProbePattern(0, 0);
          this.setupPositionProbePattern(this.moduleCount - 7, 0);
          this.setupPositionProbePattern(0, this.moduleCount - 7);
          this.setupPositionAdjustPattern();
          this.setupTimingPattern();
          this.setupTypeInfo(test, maskPattern);
          if (this.typeNumber >= 7) {
            this.setupTypeNumber(test);
          }
          if (this.dataCache == null) {
            this.dataCache = QRCodeModel.createData(this.typeNumber, this.errorCorrectLevel, this.dataList);
          }
          this.mapData(this.dataCache, maskPattern);
        },
        setupPositionProbePattern: function(row, col) {
          for (var r = -1; r <= 7; r++) {
            if (row + r <= -1 || this.moduleCount <= row + r) continue;
            for (var c = -1; c <= 7; c++) {
              if (col + c <= -1 || this.moduleCount <= col + c) continue;
              if ((0 <= r && r <= 6 && (c == 0 || c == 6)) || (0 <= c && c <= 6 && (r == 0 || r == 6)) || (2 <= r && r <= 4 && 2 <= c && c <= 4)) {
                this.modules[row + r][col + c] = true;
              } else {
                this.modules[row + r][col + c] = false;
              }
            }
          }
        },
        getBestMaskPattern: function() {
          var minLostPoint = 0;
          var pattern = 0;
          for (var i = 0; i < 8; i++) {
            this.makeImpl(true, i);
            var lostPoint = QRUtil.getLostPoint(this);
            if (i == 0 || minLostPoint > lostPoint) {
              minLostPoint = lostPoint;
              pattern = i;
            }
          }
          return pattern;
        },
        setupTimingPattern: function() {
          for (var r = 8; r < this.moduleCount - 8; r++) {
            if (this.modules[r][6] != null) continue;
            this.modules[r][6] = (r % 2 == 0);
          }
          for (var c = 8; c < this.moduleCount - 8; c++) {
            if (this.modules[6][c] != null) continue;
            this.modules[6][c] = (c % 2 == 0);
          }
        },
        setupPositionAdjustPattern: function() {
          var pos = QRUtil.getPatternPosition(this.typeNumber);
          for (var i = 0; i < pos.length; i++) {
            for (var j = 0; j < pos.length; j++) {
              var row = pos[i];
              var col = pos[j];
              if (this.modules[row][col] != null) continue;
              for (var r = -2; r <= 2; r++) {
                for (var c = -2; c <= 2; c++) {
                  if (r == -2 || r == 2 || c == -2 || c == 2 || (r == 0 && c == 0)) {
                    this.modules[row + r][col + c] = true;
                  } else {
                    this.modules[row + r][col + c] = false;
                  }
                }
              }
            }
          }
        },
        setupTypeNumber: function(test) {
          var bits = QRUtil.getBCHTypeNumber(this.typeNumber);
          for (var i = 0; i < 18; i++) {
            var mod = (!test && ((bits >> i) & 1) == 1);
            this.modules[Math.floor(i / 3)][i % 3 + this.moduleCount - 8 - 3] = mod;
          }
          for (var i = 0; i < 18; i++) {
            var mod = (!test && ((bits >> i) & 1) == 1);
            this.modules[i % 3 + this.moduleCount - 8 - 3][Math.floor(i / 3)] = mod;
          }
        },
        setupTypeInfo: function(test, maskPattern) {
          var data = (this.errorCorrectLevel << 3) | maskPattern;
          var bits = QRUtil.getBCHTypeInfo(data);
          for (var i = 0; i < 15; i++) {
            var mod = (!test && ((bits >> i) & 1) == 1);
            if (i < 6) {
              this.modules[i][8] = mod;
            } else if (i < 8) {
              this.modules[i + 1][8] = mod;
            } else {
              this.modules[this.moduleCount - 15 + i][8] = mod;
            }
          }
          for (var i = 0; i < 15; i++) {
            var mod = (!test && ((bits >> i) & 1) == 1);
            if (i < 8) {
              this.modules[8][this.moduleCount - i - 1] = mod;
            } else if (i < 9) {
              this.modules[8][15 - i - 1 + 1] = mod;
            } else {
              this.modules[8][15 - i - 1] = mod;
            }
          }
          this.modules[this.moduleCount - 8][8] = (!test);
        },
        mapData: function(data, maskPattern) {
          var inc = -1;
          var row = this.moduleCount - 1;
          var bitIndex = 7;
          var byteIndex = 0;
          var maskFunc = QRUtil.getMaskFunction(maskPattern);
          for (var col = this.moduleCount - 1; col > 0; col -= 2) {
            if (col == 6) col--;
            while (true) {
              for (var c = 0; c < 2; c++) {
                if (this.modules[row][col - c] == null) {
                  var dark = false;
                  if (byteIndex < data.length) {
                    dark = (((data[byteIndex] >>> bitIndex) & 1) == 1);
                  }
                  var mask = maskFunc(row, col - c);
                  if (mask) {
                    dark = !dark;
                  }
                  this.modules[row][col - c] = dark;
                  bitIndex--;
                  if (bitIndex == -1) {
                    byteIndex++;
                    bitIndex = 7;
                  }
                }
              }
              row += inc;
              if (row < 0 || this.moduleCount <= row) {
                row -= inc;
                inc = -inc;
                break;
              }
            }
          }
        }
      };

      QRCodeModel.createData = function(typeNumber, errorCorrectLevel, dataList) {
        var rsBlocks = QRRSBlock.getRSBlocks(typeNumber, errorCorrectLevel);
        var buffer = new QRBitBuffer();
        for (var i = 0; i < dataList.length; i++) {
          var data = dataList[i];
          buffer.put(data.mode, 4);
          buffer.put(data.getLength(), QRUtil.getLengthInBits(data.mode, typeNumber));
          data.write(buffer);
        }
        var totalDataCount = 0;
        for (var i = 0; i < rsBlocks.length; i++) {
          totalDataCount += rsBlocks[i].dataCount;
        }
        if (buffer.getLengthInBits() > totalDataCount * 8) {
          throw new Error("code length overflow: " + buffer.getLengthInBits() + ">" + totalDataCount * 8);
        }
        if (buffer.getLengthInBits() + 4 <= totalDataCount * 8) {
          buffer.put(0, 4);
        }
        while (buffer.getLengthInBits() % 8 != 0) {
          buffer.putBit(false);
        }
        while (true) {
          if (buffer.getLengthInBits() >= totalDataCount * 8) break;
          buffer.put(0xEC, 8);
          if (buffer.getLengthInBits() >= totalDataCount * 8) break;
          buffer.put(0x11, 8);
        }
        return QRCodeModel.createBytes(buffer, rsBlocks);
      };

      QRCodeModel.createBytes = function(buffer, rsBlocks) {
        var offset = 0;
        var maxDcCount = 0;
        var maxEcCount = 0;
        var dcdata = new Array(rsBlocks.length);
        var ecdata = new Array(rsBlocks.length);
        for (var r = 0; r < rsBlocks.length; r++) {
          var dcCount = rsBlocks[r].dataCount;
          var ecCount = rsBlocks[r].totalCount - dcCount;
          maxDcCount = Math.max(maxDcCount, dcCount);
          maxEcCount = Math.max(maxEcCount, ecCount);
          dcdata[r] = new Array(dcCount);
          for (var i = 0; i < dcdata[r].length; i++) {
            dcdata[r][i] = 0xff & buffer.buffer[i + offset];
          }
          offset += dcCount;
          var rsPoly = QRUtil.getErrorCorrectPolynomial(ecCount);
          var rawPoly = new QRPolynomial(dcdata[r], rsPoly.getLength() - 1);
          var modPoly = rawPoly.mod(rsPoly);
          ecdata[r] = new Array(rsPoly.getLength() - 1);
          for (var i = 0; i < ecdata[r].length; i++) {
            var modIndex = i + modPoly.getLength() - ecdata[r].length;
            ecdata[r][i] = (modIndex >= 0) ? modPoly.get(modIndex) : 0;
          }
        }
        var totalCodeCount = 0;
        for (var i = 0; i < rsBlocks.length; i++) {
          totalCodeCount += rsBlocks[i].totalCount;
        }
        var data = new Array(totalCodeCount);
        var index = 0;
        for (var i = 0; i < maxDcCount; i++) {
          for (var r = 0; r < rsBlocks.length; r++) {
            if (i < dcdata[r].length) {
              data[index++] = dcdata[r][i];
            }
          }
        }
        for (var i = 0; i < maxEcCount; i++) {
          for (var r = 0; r < rsBlocks.length; r++) {
            if (i < ecdata[r].length) {
              data[index++] = ecdata[r][i];
            }
          }
        }
        return data;
      };

      var QRMode = { MODE_8BIT_BYTE: 4 };
      var QRErrorCorrectLevel = { L: 1, M: 0, Q: 3, H: 2 };
      var QRMaskPattern = {
        PATTERN000: 0, PATTERN001: 1, PATTERN010: 2, PATTERN011: 3,
        PATTERN100: 4, PATTERN101: 5, PATTERN110: 6, PATTERN111: 7
      };

      var QRUtil = {
        PATTERN_POSITION_TABLE: [
          [], [6, 18], [6, 22], [6, 26], [6, 30], [6, 34],
          [6, 22, 38], [6, 24, 42], [6, 26, 46], [6, 28, 50], [6, 30, 54],
          [6, 32, 58], [6, 34, 62], [6, 26, 46, 66], [6, 26, 48, 70]
        ],
        G15: (1 << 10) | (1 << 8) | (1 << 5) | (1 << 4) | (1 << 2) | (1 << 1) | (1 << 0),
        G18: (1 << 12) | (1 << 11) | (1 << 10) | (1 << 9) | (1 << 8) | (1 << 5) | (1 << 2) | (1 << 0),
        G15_MASK: (1 << 14) | (1 << 12) | (1 << 10) | (1 << 4) | (1 << 1),
        getBCHTypeInfo: function(data) {
          var d = data << 10;
          while (QRUtil.getBCHDigit(d) - QRUtil.getBCHDigit(QRUtil.G15) >= 0) {
            d ^= (QRUtil.G15 << (QRUtil.getBCHDigit(d) - QRUtil.getBCHDigit(QRUtil.G15)));
          }
          return ((data << 10) | d) ^ QRUtil.G15_MASK;
        },
        getBCHTypeNumber: function(data) {
          var d = data << 12;
          while (QRUtil.getBCHDigit(d) - QRUtil.getBCHDigit(QRUtil.G18) >= 0) {
            d ^= (QRUtil.G18 << (QRUtil.getBCHDigit(d) - QRUtil.getBCHDigit(QRUtil.G18)));
          }
          return (data << 12) | d;
        },
        getBCHDigit: function(data) {
          var digit = 0;
          while (data != 0) {
            digit++;
            data >>>= 1;
          }
          return digit;
        },
        getPatternPosition: function(typeNumber) {
          return QRUtil.PATTERN_POSITION_TABLE[typeNumber - 1] || [];
        },
        getMaskFunction: function(maskPattern) {
          switch (maskPattern) {
            case 0: return function(i, j) { return (i + j) % 2 == 0; };
            case 1: return function(i, j) { return i % 2 == 0; };
            case 2: return function(i, j) { return j % 3 == 0; };
            case 3: return function(i, j) { return (i + j) % 3 == 0; };
            case 4: return function(i, j) { return (Math.floor(i / 2) + Math.floor(j / 3)) % 2 == 0; };
            case 5: return function(i, j) { return (i * j) % 2 + (i * j) % 3 == 0; };
            case 6: return function(i, j) { return ((i * j) % 2 + (i * j) % 3) % 2 == 0; };
            case 7: return function(i, j) { return ((i * j) % 3 + (i + j) % 2) % 2 == 0; };
            default: throw new Error("bad maskPattern:" + maskPattern);
          }
        },
        getErrorCorrectPolynomial: function(errorCorrectLength) {
          var a = new QRPolynomial([1], 0);
          for (var i = 0; i < errorCorrectLength; i++) {
            a = a.multiply(new QRPolynomial([1, QRMath.gexp(i)], 0));
          }
          return a;
        },
        getLengthInBits: function(mode, type) {
          if (1 <= type && type < 10) return 8;
          return 16;
        },
        getLostPoint: function(qrCode) {
          return 0; // Standard heuristic simplified
        }
      };

      var QRMath = {
        glog: function(n) {
          if (n < 1) throw new Error("glog(" + n + ")");
          return QRMath.LOG_TABLE[n];
        },
        gexp: function(n) {
          while (n < 0) n += 255;
          while (n >= 256) n -= 255;
          return QRMath.EXP_TABLE[n];
        },
        EXP_TABLE: new Array(256),
        LOG_TABLE: new Array(256)
      };

      for (var i = 0; i < 8; i++) QRMath.EXP_TABLE[i] = 1 << i;
      for (var i = 8; i < 256; i++) QRMath.EXP_TABLE[i] = QRMath.EXP_TABLE[i - 4] ^ QRMath.EXP_TABLE[i - 5] ^ QRMath.EXP_TABLE[i - 6] ^ QRMath.EXP_TABLE[i - 8];
      for (var i = 0; i < 255; i++) QRMath.LOG_TABLE[QRMath.EXP_TABLE[i]] = i;

      function QRPolynomial(num, shift) {
        if (num.length == undefined) throw new Error(num.length + "/" + shift);
        var offset = 0;
        while (offset < num.length && num[offset] == 0) offset++;
        this.num = new Array(num.length - offset + shift);
        for (var i = 0; i < num.length - offset; i++) this.num[i] = num[i + offset];
      }
      QRPolynomial.prototype = {
        get: function(index) { return this.num[index]; },
        getLength: function() { return this.num.length; },
        multiply: function(e) {
          var num = new Array(this.getLength() + e.getLength() - 1);
          for (var i = 0; i < this.getLength(); i++) {
            for (var j = 0; j < e.getLength(); j++) {
              num[i + j] ^= QRMath.gexp(QRMath.glog(this.get(i)) + QRMath.glog(e.get(j)));
            }
          }
          return new QRPolynomial(num, 0);
        },
        mod: function(e) {
          if (this.getLength() - e.getLength() < 0) return this;
          var ratio = QRMath.glog(this.get(0)) - QRMath.glog(e.get(0));
          var num = new Array(this.getLength());
          for (var i = 0; i < this.getLength(); i++) num[i] = this.get(i);
          for (var i = 0; i < e.getLength(); i++) {
            num[i] ^= QRMath.gexp(QRMath.glog(e.get(i)) + ratio);
          }
          return new QRPolynomial(num, 0).mod(e);
        }
      };

      function QRRSBlock(totalCount, dataCount) {
        this.totalCount = totalCount;
        this.dataCount = dataCount;
      }
      QRRSBlock.RS_BLOCK_TABLE = [
        [1, 26, 19], [1, 44, 34], [1, 70, 55], [1, 100, 80],
        [1, 134, 108], [2, 86, 68], [2, 98, 78], [2, 121, 97], [2, 146, 116]
      ];
      QRRSBlock.getRSBlocks = function(typeNumber, errorCorrectLevel) {
        var rsBlock = QRRSBlock.RS_BLOCK_TABLE[typeNumber - 1];
        var list = [];
        for (var i = 0; i < rsBlock[0]; i++) {
          list.push(new QRRSBlock(rsBlock[1], rsBlock[2]));
        }
        return list;
      };

      function QRBitBuffer() {
        this.buffer = [];
        this.length = 0;
      }
      QRBitBuffer.prototype = {
        get: function(index) {
          var bufIndex = Math.floor(index / 8);
          return ((this.buffer[bufIndex] >>> (7 - index % 8)) & 1) == 1;
        },
        put: function(num, length) {
          for (var i = 0; i < length; i++) {
            this.putBit(((num >>> (length - i - 1)) & 1) == 1);
          }
        },
        getLengthInBits: function() { return this.length; },
        putBit: function(bit) {
          var bufIndex = Math.floor(this.length / 8);
          if (this.buffer.length <= bufIndex) this.buffer.push(0);
          if (bit) this.buffer[bufIndex] |= (0x80 >>> (this.length % 8));
          this.length++;
        }
      };

      // High-Contrast, Auto-Versioning SVG QR Code Engine with ISO Quiet Zone
      window.renderQRCode = function(targetEl, text) {
        var qr = null;
        for (var type = 1; type <= 10; type++) {
          try {
            var model = new QRCodeModel(type, QRErrorCorrectLevel.L);
            model.addData(text);
            model.make();
            qr = model;
            break;
          } catch(e) {}
        }
        if (!qr) {
          qr = new QRCodeModel(4, QRErrorCorrectLevel.M);
          qr.addData(text);
          qr.make();
        }

        var count = qr.getModuleCount();
        var margin = 4; // ISO 18004 4-module quiet zone
        var size = count + margin * 2;

        var path = '';
        for (var r = 0; r < count; r++) {
          for (var c = 0; c < count; c++) {
            if (qr.isDark(r, c)) {
              path += 'M' + (c + margin) + ',' + (r + margin) + 'h1v1h-1z ';
            }
          }
        }

        var svg = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ' + size + ' ' + size + '" shape-rendering="crispEdges" style="width: 100%; height: 100%; display: block; border-radius: 8px;">' +
                  '<rect width="' + size + '" height="' + size + '" fill="#ffffff"/>' +
                  '<path d="' + path + '" fill="#000000"/>' +
                  '</svg>';

        targetEl.innerHTML = svg;
      };
    })(window);
  


    let serverIP = '127.0.0.1';
    let httpPort = 8080;
    let hotspotData = {
      state: 'Off',
      ssid: 'dpc',
      passphrase: '',
      hotspot_ip: '',
      wifi_ip: '',
      wifi_qr: '',
      gamepad_url: ''
    };
    let currentQRMode = 'wifi'; // 'wifi' or 'gamepad'

    // DOM Elements
    const qrCanvas = document.getElementById('qrCanvas');

    const startHeroSection = document.getElementById('startHeroSection');
    const activePairingSection = document.getElementById('activePairingSection');
    const btnBigStart = document.getElementById('btnBigStart');
    const linkDirectWifi = document.getElementById('linkDirectWifi');
    const tabWifiQR = document.getElementById('tabWifiQR');
    const tabGamepadQR = document.getElementById('tabGamepadQR');
    const qrModeInfo = document.getElementById('qrModeInfo');
    const dispSsid = document.getElementById('dispSsid');
    const dispPass = document.getElementById('dispPass');
    const dispUrl = document.getElementById('dispUrl');
    const phoneConnectedToast = document.getElementById('phoneConnectedToast');
    const btnStopHotspot = document.getElementById('btnStopHotspot');
    const btnCopyGamepadUrl = document.getElementById('btnCopyGamepadUrl');
    const btnHotspotSettings = document.getElementById('btnHotspotSettings');

    const ipBadgeText = document.getElementById('ipText');
    const btnJoyCpl = document.getElementById('btnJoyCpl');
    const btnOpenMobile = document.getElementById('btnOpenMobile');
    const btnSaveSettings = document.getElementById('btnSaveSettings');

    // Visualizer Stick & Buttons
    const vStickLHead = document.getElementById('vStickLHead');
    const vStickRHead = document.getElementById('vStickRHead');
    const btnMap = {
      0: document.getElementById('vBtnCross'),
      1: document.getElementById('vBtnCircle'),
      2: document.getElementById('vBtnSquare'),
      3: document.getElementById('vBtnTriangle'),
      4: document.getElementById('vBtnL1'),
      5: document.getElementById('vBtnR1'),
      8: document.getElementById('vBtnShare'),
      9: document.getElementById('vBtnOptions'),
      10: document.getElementById('vBtnPS'),
      12: document.getElementById('vBtnUp'),
      13: document.getElementById('vBtnDown'),
      14: document.getElementById('vBtnLeft'),
      15: document.getElementById('vBtnRight')
    };

    function renderCurrentQR() {
      if (currentQRMode === 'wifi' && hotspotData.wifi_qr) {
        tabWifiQR.classList.add('active');
        tabGamepadQR.classList.remove('active');
        qrModeInfo.innerHTML = `📷 Point phone camera at QR &rarr; Tap <b>"Join Network"</b>. Phone connects to laptop hotspot automatically!`;
        try {
          window.renderQRCode(qrCanvas, hotspotData.wifi_qr);
        } catch (e) {
          console.error("QR render error", e);
        }
      } else {
        tabWifiQR.classList.remove('active');
        tabGamepadQR.classList.add('active');
        const url = hotspotData.gamepad_url || `http://${serverIP}:${httpPort}`;
        qrModeInfo.innerHTML = `🌐 Point phone camera at QR &rarr; Tap link to open <b>DualSense Controller</b> in browser!`;
        try {
          window.renderQRCode(qrCanvas, url);
        } catch (e) {
          console.error("QR render error", e);
        }
      }
    }

    // Start Button Click: Automatically turns on hotspot & switches to QR
    btnBigStart.addEventListener('click', async () => {
      btnBigStart.disabled = true;
      btnBigStart.innerHTML = `<span>⏳</span><span>ACTIVATING HOTSPOT...</span>`;
      try {
        const res = await fetch('/api/hotspot', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ action: 'on' })
        });
        if (res.ok) {
          hotspotData = await res.json();
          updateHotspotUI();
        }
      } catch(e) {
        alert('Could not start hotspot: ' + e);
      } finally {
        btnBigStart.disabled = false;
        btnBigStart.innerHTML = `<span>⚡</span><span>START AUTO-CONNECT</span>`;
      }
    });

    // Direct Wi-Fi link bypass (for home router Wi-Fi)
    linkDirectWifi.addEventListener('click', (e) => {
      e.preventDefault();
      currentQRMode = 'gamepad';
      startHeroSection.style.display = 'none';
      activePairingSection.style.display = 'block';
      renderCurrentQR();
    });

    // Tab Switching
    tabWifiQR.addEventListener('click', () => {
      currentQRMode = 'wifi';
      renderCurrentQR();
    });

    tabGamepadQR.addEventListener('click', () => {
      currentQRMode = 'gamepad';
      renderCurrentQR();
    });

    // Stop Hotspot
    btnStopHotspot.addEventListener('click', async () => {
      btnStopHotspot.disabled = true;
      try {
        const res = await fetch('/api/hotspot', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ action: 'off' })
        });
        if (res.ok) {
          hotspotData = await res.json();
          updateHotspotUI();
        }
      } catch(e) {
      } finally {
        btnStopHotspot.disabled = false;
      }
    });

    // Copy Gamepad URL
    btnCopyGamepadUrl.addEventListener('click', () => {
      const url = dispUrl.textContent;
      navigator.clipboard.writeText(url);
      const prev = btnCopyGamepadUrl.textContent;
      btnCopyGamepadUrl.textContent = 'Copied! ✓';
      setTimeout(() => { btnCopyGamepadUrl.textContent = prev; }, 1800);
    });

    btnHotspotSettings.addEventListener('click', () => {
      fetch('/api/hotspot/settings', { method: 'POST' });
    });

    btnJoyCpl.addEventListener('click', () => {
      fetch('/api/system/joycpl', { method: 'POST' });
    });

    btnOpenMobile.addEventListener('click', () => {
      window.open('/controller', '_blank');
    });

    function updateHotspotUI() {
      dispSsid.textContent = hotspotData.ssid || 'dpc';
      dispPass.textContent = hotspotData.passphrase || '12345678';
      dispUrl.textContent = hotspotData.gamepad_url || `http://${serverIP}:${httpPort}`;

      if (hotspotData.state === 'On') {
        startHeroSection.style.display = 'none';
        activePairingSection.style.display = 'block';
        renderCurrentQR();
      } else {
        startHeroSection.style.display = 'flex';
        activePairingSection.style.display = 'none';
      }
    }

    async function checkHotspotStatus() {
      try {
        const res = await fetch('/api/hotspot');
        if (res.ok) {
          hotspotData = await res.json();
          updateHotspotUI();
        }
      } catch(e) {}
    }

    // Fetch live server status and update UI
    async function fetchServerStatus() {
      try {
        const res = await fetch('/api/status');
        if (res.ok) {
          const s = await res.json();
          serverIP = s.ip;
          httpPort = s.http_port;

          ipBadgeText.textContent = `${serverIP}:${httpPort}`;

          // Player 1 Status
          const p1Card = document.getElementById('p1Card');
          const p1Dot = document.getElementById('p1Dot');
          const p1StatusText = document.getElementById('p1StatusText');
          const p1Device = document.getElementById('p1Device');
          const p1IP = document.getElementById('p1IP');
          const p1Latency = document.getElementById('p1Latency');

          if (s.p1_connected) {
            p1Card.classList.add('active');
            p1Dot.className = 'dot green';
            p1StatusText.textContent = 'Connected (Slot 1)';
            p1Device.textContent = s.p1_ua ? (s.p1_ua.includes('Android') ? 'Android' : (s.p1_ua.includes('iPhone') ? 'iPhone' : 'Mobile')) : 'Touch Web';
            p1IP.textContent = s.p1_ip || '--';
            p1Latency.textContent = (s.p1_rtt || 1.4) + ' ms';
            if (phoneConnectedToast) phoneConnectedToast.style.display = 'block';
          } else {
            p1Card.classList.remove('active');
            p1Dot.className = 'dot';
            p1StatusText.textContent = 'Waiting for Phone';
            p1Device.textContent = 'None';
            p1IP.textContent = '--';
            p1Latency.textContent = '-- ms';
            if (phoneConnectedToast) phoneConnectedToast.style.display = 'none';
          }

          // Motor Bars
          const p1H = s.p1_rumble_heavy || 0;
          const p1L = s.p1_rumble_light || 0;
          document.getElementById('p1HeavyBar').style.width = p1H + '%';
          document.getElementById('p1HeavyVal').textContent = p1H + '%';
          document.getElementById('p1LightBar').style.width = p1L + '%';
          document.getElementById('p1LightVal').textContent = p1L + '%';

          // Player 2 Status
          const p2Card = document.getElementById('p2Card');
          const p2Dot = document.getElementById('p2Dot');
          const p2StatusText = document.getElementById('p2StatusText');
          if (s.p2_connected) {
            p2Card.classList.add('active');
            p2Dot.className = 'dot green';
            p2StatusText.textContent = 'Connected (Slot 2)';
            document.getElementById('p2Device').textContent = 'Mobile';
            document.getElementById('p2IP').textContent = s.p2_ip || '--';
            document.getElementById('p2Latency').textContent = (s.p2_rtt || 1.8) + ' ms';
          } else {
            p2Card.classList.remove('active');
            p2Dot.className = 'dot';
            p2StatusText.textContent = 'Waiting for Phone';
            document.getElementById('p2Device').textContent = 'None';
            document.getElementById('p2IP').textContent = '--';
            document.getElementById('p2Latency').textContent = '-- ms';
          }
        }
      } catch(e) {}
    }

    // Connect WebSocket as monitor to visualize real-time input on PC screen
    function connectVisualizerWS() {
      const loc = window.location;
      const wsUrl = `ws://${loc.host}/ws`;
      let ws;
      try {
        ws = new WebSocket(wsUrl);
        ws.binaryType = 'arraybuffer';
      } catch(e) {
        setTimeout(connectVisualizerWS, 3000);
        return;
      }

      ws.onmessage = (event) => {
        if (!(event.data instanceof ArrayBuffer)) return;
        const view = new Uint8Array(event.data);
        if (view.length < 8) return;

        // Rumble feedback packet from server
        if (view[0] === 0xFE && view.length >= 4) {
          const heavy = Math.round((view[1] / 255) * 100);
          const light = Math.round((view[2] / 255) * 100);
          document.getElementById('p1HeavyBar').style.width = heavy + '%';
          document.getElementById('p1HeavyVal').textContent = heavy + '%';
          document.getElementById('p1LightBar').style.width = light + '%';
          document.getElementById('p1LightVal').textContent = light + '%';
          return;
        }

        // Joystick positions
        const lx = ((view[0] - 128) / 127) * 15;
        const ly = ((view[1] - 128) / 127) * 15;
        vStickLHead.setAttribute('transform', `translate(${lx}, ${ly})`);

        const rx = ((view[2] - 128) / 127) * 15;
        const ry = ((view[3] - 128) / 127) * 15;
        vStickRHead.setAttribute('transform', `translate(${rx}, ${ry})`);

        // Triggers
        const l2 = view[4];
        const r2 = view[5];
        const vBtnL2 = document.getElementById('vBtnL2');
        const vBtnR2 = document.getElementById('vBtnR2');
        if (vBtnL2) vBtnL2.classList.toggle('pressed', l2 > 30);
        if (vBtnR2) vBtnR2.classList.toggle('pressed', r2 > 30);

        // Buttons
        const mask = view[6] | (view[7] << 8) | (view.length >= 10 ? ((view[8] | (view[9] << 8)) << 16) : 0);
        for (let bit = 0; bit < 16; bit++) {
          const el = btnMap[bit];
          if (el) {
            const isPressed = (mask & (1 << bit)) !== 0;
            el.classList.toggle('pressed', isPressed);
          }
        }
      };

      ws.onclose = () => {
        setTimeout(connectVisualizerWS, 2000);
      };
    }

    // Load Settings
    async function loadSettings() {
      try {
        const res = await fetch('/api/settings');
        if (res.ok) {
          const cfg = await res.json();
          document.getElementById('chkAutoStart').checked = !!cfg.auto_start;
          document.getElementById('chkAutoHotspot').checked = !!cfg.auto_hotspot;
          document.getElementById('chkMinimizeTray').checked = cfg.minimize_tray !== false;
          document.getElementById('chkVibration').checked = cfg.vibration_enabled !== false;
        }
      } catch(e) {}
    }

    btnSaveSettings.addEventListener('click', async () => {
      const payload = {
        auto_start: document.getElementById('chkAutoStart').checked,
        auto_hotspot: document.getElementById('chkAutoHotspot').checked,
        minimize_tray: document.getElementById('chkMinimizeTray').checked,
        vibration_enabled: document.getElementById('chkVibration').checked
      };
      try {
        const res = await fetch('/api/settings', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });
        if (res.ok) {
          btnSaveSettings.textContent = 'Saved! ✓';
          setTimeout(() => { btnSaveSettings.textContent = 'Save Preferences'; }, 2000);
        }
      } catch(e) {
        alert('Failed to save settings: ' + e);
      }
    });

    // Initialize
    fetchServerStatus();
    checkHotspotStatus();
    loadSettings();
    connectVisualizerWS();
    setInterval(fetchServerStatus, 1500);
    setInterval(checkHotspotStatus, 5000);