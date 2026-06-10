import Cocoa
import Darwin

class AppDelegate: NSObject, NSApplicationDelegate {
    var statusItem: NSStatusItem!
    var serverTask: Process?
    var serverPort: Int = 3456
    var isRunning = false
    var logFile: FileHandle?
    var autoSyncTimer: Timer?
    var restartAttempts = 0
    var maxRestartAttempts = 10
    
    // MARK: - Lifecycle
    
    func applicationDidFinishLaunching(_ notification: Notification) {
        setupStatusItem()
        updateMenu()
        startServer()
    }
    
    func applicationWillTerminate(_ notification: Notification) {
        stopAutoSync()
        stopServer()
    }
    
    // MARK: - UI Setup
    
    private func setupStatusItem() {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        statusItem.button?.title = "📡"
        
        // Enable right-click by adding a target/action to the button
        if let button = statusItem.button {
            button.target = self
            button.action = #selector(statusItemClicked(_:))
            button.sendAction(on: [.leftMouseUp, .rightMouseUp])
        }
        
        buildMenu()
    }
    
    private func buildMenu() {
        let menu = NSMenu()
        
        let titleItem = NSMenuItem(title: "Lark Radar", action: nil, keyEquivalent: "")
        titleItem.isEnabled = false
        menu.addItem(titleItem)
        
        let statusMenuItem = NSMenuItem(title: "Status: Stopped", action: nil, keyEquivalent: "")
        statusMenuItem.tag = 1
        statusMenuItem.isEnabled = false
        menu.addItem(statusMenuItem)
        
        menu.addItem(NSMenuItem.separator())
        
        let syncItem = NSMenuItem(title: "🔄  Sync Now", action: #selector(triggerSync), keyEquivalent: "s")
        syncItem.tag = 2
        menu.addItem(syncItem)
        
        let toggleItem = NSMenuItem(title: "▶️  Start Server", action: #selector(toggleServer), keyEquivalent: "t")
        toggleItem.tag = 100
        menu.addItem(toggleItem)
        
        menu.addItem(NSMenuItem.separator())
        
        // Quick links section
        let linksItem = NSMenuItem(title: "Quick Links", action: nil, keyEquivalent: "")
        linksItem.isEnabled = false
        menu.addItem(linksItem)
        
        let dashboardItem = NSMenuItem(title: "📊  Open Dashboard...", action: #selector(openDashboard), keyEquivalent: "o")
        dashboardItem.tag = 3
        menu.addItem(dashboardItem)
        
        let dataDirItem = NSMenuItem(title: "📁  Open Data Directory", action: #selector(openDataDirectory), keyEquivalent: "d")
        menu.addItem(dataDirItem)
        
        let logsItem = NSMenuItem(title: "📋  Open Logs", action: #selector(openLogs), keyEquivalent: "l")
        menu.addItem(logsItem)
        
        menu.addItem(NSMenuItem.separator())
        
        let quitItem = NSMenuItem(title: "Quit", action: #selector(quitApp), keyEquivalent: "q")
        menu.addItem(quitItem)
        
        statusItem.menu = menu
    }
    
    @objc private func statusItemClicked(_ sender: NSStatusBarButton) {
        guard let event = NSApp.currentEvent else { return }
        
        if event.type == .rightMouseUp {
            // Right-click: show context menu without left-click menu
            showContextMenu()
        } else {
            // Left-click: show normal menu
            statusItem.button?.performClick(nil)
        }
    }
    
    private func showContextMenu() {
        let menu = NSMenu()
        
        // Status header
        let statusHeader = NSMenuItem(title: isRunning ? "🟢 Running on :\(serverPort)" : "⚫ Stopped", action: nil, keyEquivalent: "")
        statusHeader.isEnabled = false
        menu.addItem(statusHeader)
        
        menu.addItem(NSMenuItem.separator())
        
        // Quick actions
        if isRunning {
            let syncItem = NSMenuItem(title: "🔄 Sync Now", action: #selector(triggerSync), keyEquivalent: "")
            menu.addItem(syncItem)
            
            let dashboardItem = NSMenuItem(title: "📊 Dashboard", action: #selector(openDashboard), keyEquivalent: "")
            menu.addItem(dashboardItem)
            
            let stopItem = NSMenuItem(title: "⏹  Stop Server", action: #selector(toggleServer), keyEquivalent: "")
            menu.addItem(stopItem)
        } else {
            let startItem = NSMenuItem(title: "▶️  Start Server", action: #selector(toggleServer), keyEquivalent: "")
            menu.addItem(startItem)
        }
        
        menu.addItem(NSMenuItem.separator())
        
        // Utility actions
        let dataDirItem = NSMenuItem(title: "📁 Data Directory", action: #selector(openDataDirectory), keyEquivalent: "")
        menu.addItem(dataDirItem)
        
        let logsItem = NSMenuItem(title: "📋 Logs", action: #selector(openLogs), keyEquivalent: "")
        menu.addItem(logsItem)
        
        menu.addItem(NSMenuItem.separator())
        
        let quitItem = NSMenuItem(title: "Quit", action: #selector(quitApp), keyEquivalent: "")
        menu.addItem(quitItem)
        
        // Show menu at status item location
        if let button = statusItem.button {
            menu.popUp(positioning: nil, at: NSPoint(x: 0, y: button.bounds.height + 4), in: button)
        }
    }
    
    private func updateMenu() {
        guard let menu = statusItem.menu else { return }
        
        if let statusMenuItem = menu.item(withTag: 1) {
            statusMenuItem.title = isRunning
                ? "Status: Running on :\(serverPort)"
                : "Status: Stopped"
        }
        
        if let toggleItem = menu.item(withTag: 100) {
            toggleItem.title = isRunning ? "⏹  Stop Server" : "▶️  Start Server"
        }
        
        if let syncItem = menu.item(withTag: 2) {
            syncItem.isEnabled = isRunning
        }
        
        if let dashboardItem = menu.item(withTag: 3) {
            dashboardItem.isEnabled = isRunning
        }
        
        statusItem.button?.title = isRunning ? "🟢" : "📡"
    }
    
    // MARK: - Server Management
    
    @objc func toggleServer() {
        if isRunning {
            stopServer()
        } else {
            startServer()
        }
    }
    
    func startServer() {
        guard !isRunning else { return }
        
        let resourcesPath = Bundle.main.resourcePath!
        let goBinaryPath = resourcesPath + "/lark-radar-server"
        let dataDir = NSHomeDirectory() + "/.lark-radar"
        
        let fm = FileManager.default
        guard fm.fileExists(atPath: goBinaryPath) else {
            showAlert(message: "Go data service not found in app bundle. Expected: \(goBinaryPath)")
            return
        }
        
        serverPort = readPortFromConfig(dataDir)
        
        let task = Process()
        task.executableURL = URL(fileURLWithPath: goBinaryPath)
        task.arguments = ["--port", "\(serverPort)", "--data-dir", dataDir]
        task.environment = [
            "LARK_RADAR_DATA_DIR": dataDir,
            "LARK_RADAR_PORT": "\(serverPort)"
        ]
        
        let logPath = dataDir + "/server.log"
        try? fm.createDirectory(atPath: dataDir, withIntermediateDirectories: true, attributes: nil)
        if fm.fileExists(atPath: logPath) {
            // Rotate log: keep last 5MB, truncate current
            let attr = try? fm.attributesOfItem(atPath: logPath)
            if let size = attr?[.size] as? UInt64, size > 5 * 1024 * 1024 {
                try? fm.removeItem(atPath: logPath)
            }
        }
        fm.createFile(atPath: logPath, contents: nil, attributes: nil)
        if let logHandle = FileHandle(forWritingAtPath: logPath) {
            // Append mode
            _ = try? logHandle.seekToEnd()
            task.standardOutput = logHandle
            task.standardError = logHandle
            logFile = logHandle
        }
        
        task.terminationHandler = { [weak self] _ in
            DispatchQueue.main.async {
                self?.handleServerTermination()
            }
        }
        
        do {
            try task.run()
            serverTask = task
            isRunning = true
            restartAttempts = 0
            updateMenu()
            
            // Start auto-sync timer if configured
            startAutoSyncIfNeeded(dataDir: dataDir)
            
            // Wait for health check
            DispatchQueue.global().async { [weak self] in
                for _ in 0..<30 {
                    if self?.isServerReady() == true {
                        break
                    }
                    Thread.sleep(forTimeInterval: 1)
                }
            }
        } catch {
            showAlert(message: "Failed to start data service: \(error.localizedDescription)")
            isRunning = false
            updateMenu()
        }
    }
    
    func stopServer() {
        guard let task = serverTask else { return }
        stopAutoSync()
        task.terminate()
        // Give it 3 seconds to exit gracefully
        DispatchQueue.global().asyncAfter(deadline: .now() + 3) { [weak self] in
            if task.isRunning {
                task.interrupt()
                DispatchQueue.global().asyncAfter(deadline: .now() + 2) {
                    if task.isRunning {
                        task.terminate()
                    }
                }
            }
            DispatchQueue.main.async {
                self?.serverTask = nil
                self?.isRunning = false
                self?.updateMenu()
            }
        }
    }
    
    private func handleServerTermination() {
        guard isRunning else { return }
        isRunning = false
        serverTask = nil
        updateMenu()
        
        // Auto-restart with exponential backoff
        guard restartAttempts < maxRestartAttempts else {
            showAlert(message: "Data service failed to start after \(maxRestartAttempts) attempts. Please check logs.")
            return
        }
        
        let delay = min(5 * (1 << restartAttempts), 60)
        restartAttempts += 1
        
        DispatchQueue.main.asyncAfter(deadline: .now() + .seconds(delay)) { [weak self] in
            self?.startServer()
        }
    }
    
    // MARK: - Health Check
    
    func isServerReady() -> Bool {
        guard let url = URL(string: "http://localhost:\(serverPort)/api/health") else { return false }
        var request = URLRequest(url: url)
        request.timeoutInterval = 2
        let semaphore = DispatchSemaphore(value: 0)
        var ready = false
        let task = URLSession.shared.dataTask(with: request) { data, response, _ in
            if let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 {
                if let data = data,
                   let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                   json["ok"] as? Bool == true {
                    ready = true
                }
            }
            semaphore.signal()
        }
        task.resume()
        semaphore.wait()
        return ready
    }
    
    // MARK: - Sync
    
    @objc func triggerSync() {
        guard isRunning else {
            showAlert(message: "Data service is not running.")
            return
        }
        
        guard let url = URL(string: "http://localhost:\(serverPort)/api/sync") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["days_back": 7])
        
        let task = URLSession.shared.dataTask(with: request) { [weak self] data, response, error in
            DispatchQueue.main.async {
                if let error = error {
                    self?.showAlert(message: "Sync failed: \(error.localizedDescription)")
                    return
                }
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                    self?.showAlert(message: "Sync request failed.")
                    return
                }
                // Sync started successfully (it's async)
            }
        }
        task.resume()
    }
    
    private func startAutoSyncIfNeeded(dataDir: String) {
        let configPath = dataDir + "/config.json"
        guard let data = try? Data(contentsOf: URL(fileURLWithPath: configPath)),
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let interval = json["autoSyncInterval"] as? Int, interval > 0 else {
            return
        }
        
        stopAutoSync()
        autoSyncTimer = Timer.scheduledTimer(withTimeInterval: TimeInterval(interval * 60), repeats: true) { [weak self] _ in
            self?.triggerSync()
        }
    }
    
    private func stopAutoSync() {
        autoSyncTimer?.invalidate()
        autoSyncTimer = nil
    }
    
    // MARK: - Config
    
    private func readPortFromConfig(_ dataDir: String) -> Int {
        let configPath = dataDir + "/config.json"
        guard let data = try? Data(contentsOf: URL(fileURLWithPath: configPath)),
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let port = json["port"] as? Int else {
            return 3456
        }
        return (port >= 1024 && port <= 65535) ? port : 3456
    }
    
    // MARK: - Actions
    
    @objc func openDataDirectory() {
        let dataDir = NSHomeDirectory() + "/.lark-radar"
        let url = URL(fileURLWithPath: dataDir)
        NSWorkspace.shared.open(url)
    }
    
    @objc func openLogs() {
        let logPath = NSHomeDirectory() + "/.lark-radar/server.log"
        let url = URL(fileURLWithPath: logPath)
        NSWorkspace.shared.open(url)
    }
    
    @objc func openDashboard() {
        guard isRunning else {
            showAlert(message: "Data service is not running.")
            return
        }
        let url = URL(string: "http://localhost:\(serverPort)")!
        NSWorkspace.shared.open(url)
    }
    
    @objc func quitApp() {
        stopAutoSync()
        stopServer()
        NSApplication.shared.terminate(nil)
    }
    
    // MARK: - Utilities
    
    func showAlert(message: String) {
        let alert = NSAlert()
        alert.messageText = message
        alert.alertStyle = .critical
        alert.runModal()
    }
    
    func findAvailablePort(startingAt port: Int) -> Int {
        var p = port
        while p < 65535 {
            let fd = Darwin.socket(AF_INET, SOCK_STREAM, 0)
            guard fd >= 0 else { return p }
            var addr = sockaddr_in()
            addr.sin_family = sa_family_t(AF_INET)
            addr.sin_addr.s_addr = Darwin.inet_addr("127.0.0.1")
            addr.sin_port = UInt16(p).bigEndian
            let addrSize = MemoryLayout<sockaddr_in>.size
            let result = withUnsafePointer(to: &addr) {
                $0.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                    Darwin.bind(fd, $0, socklen_t(addrSize))
                }
            }
            Darwin.close(fd)
            if result == 0 { return p }
            p += 1
        }
        return 3456
    }
}

let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.run()
