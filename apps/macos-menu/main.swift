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
    //
    // Native menu-bar conventions:
    //   * Status item icon is a template SF Symbol — macOS recolors it for
    //     light/dark menu bar and the active accent color automatically.
    //   * Icon swaps between two symbols to reflect server state, instead
    //     of an emoji + redundant text status line.
    //   * One menu used by both left- AND right-click. Assigning
    //     `statusItem.menu` is enough; macOS pops it up for both buttons.
    //   * Menu items carry small SF Symbol icons via `image`, not emoji
    //     prefixes — this matches what e.g. Things, Cleanshot, Linear do.

    private func setupStatusItem() {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        if let button = statusItem.button {
            button.image = statusBarImage(running: false)
            button.imagePosition = .imageOnly
            button.toolTip = "Lark Radar"
        }
        buildMenu()
    }

    /// Returns the menu-bar icon for the given state, as a template image
    /// so macOS handles light/dark/accent recoloring.
    private func statusBarImage(running: Bool) -> NSImage? {
        let name = running
            ? "antenna.radiowaves.left.and.right"
            : "antenna.radiowaves.left.and.right.slash"
        guard let image = NSImage(systemSymbolName: name, accessibilityDescription: "Lark Radar") else {
            return nil
        }
        image.isTemplate = true
        return image
    }

    /// Returns a small SF Symbol scaled for menu-item leading icons.
    private func menuIcon(_ name: String) -> NSImage? {
        guard let image = NSImage(systemSymbolName: name, accessibilityDescription: nil) else {
            return nil
        }
        image.isTemplate = true
        return image
    }

    private func buildMenu() {
        let menu = NSMenu()
        menu.autoenablesItems = false

        // Disabled title row — Apple's pattern for menu-bar apps. The text
        // also acts as the "About this app" affordance via .action below.
        let titleItem = NSMenuItem(title: "Lark Radar \(versionString())", action: nil, keyEquivalent: "")
        titleItem.isEnabled = false
        menu.addItem(titleItem)

        menu.addItem(NSMenuItem.separator())

        let syncItem = NSMenuItem(title: "Sync Now", action: #selector(triggerSync), keyEquivalent: "s")
        syncItem.image = menuIcon("arrow.triangle.2.circlepath")
        syncItem.tag = 2
        menu.addItem(syncItem)

        let toggleItem = NSMenuItem(title: "Start Server", action: #selector(toggleServer), keyEquivalent: "t")
        toggleItem.image = menuIcon("play.fill")
        toggleItem.tag = 100
        menu.addItem(toggleItem)

        menu.addItem(NSMenuItem.separator())

        let dashboardItem = NSMenuItem(title: "Open Dashboard", action: #selector(openDashboard), keyEquivalent: "o")
        dashboardItem.image = menuIcon("safari")
        dashboardItem.tag = 3
        menu.addItem(dashboardItem)

        let dataDirItem = NSMenuItem(title: "Open Data Folder", action: #selector(openDataDirectory), keyEquivalent: "d")
        dataDirItem.image = menuIcon("folder")
        menu.addItem(dataDirItem)

        let logsItem = NSMenuItem(title: "Open Logs", action: #selector(openLogs), keyEquivalent: "l")
        logsItem.image = menuIcon("doc.text")
        menu.addItem(logsItem)

        menu.addItem(NSMenuItem.separator())

        let aboutItem = NSMenuItem(title: "About Lark Radar", action: #selector(showAbout), keyEquivalent: "")
        menu.addItem(aboutItem)

        let quitItem = NSMenuItem(title: "Quit Lark Radar", action: #selector(quitApp), keyEquivalent: "q")
        menu.addItem(quitItem)

        statusItem.menu = menu
    }

    private func updateMenu() {
        guard let menu = statusItem.menu else { return }

        if let toggleItem = menu.item(withTag: 100) {
            toggleItem.title = isRunning ? "Stop Server" : "Start Server"
            toggleItem.image = menuIcon(isRunning ? "stop.fill" : "play.fill")
        }

        if let syncItem = menu.item(withTag: 2) {
            syncItem.isEnabled = isRunning
        }

        if let dashboardItem = menu.item(withTag: 3) {
            dashboardItem.isEnabled = isRunning
        }

        // Reflect state in the menu-bar icon itself; the text status line
        // that used to live inside the menu is no longer needed.
        statusItem.button?.image = statusBarImage(running: isRunning)
        statusItem.button?.toolTip = isRunning
            ? "Lark Radar — running on :\(serverPort)"
            : "Lark Radar — stopped"
    }

    private func versionString() -> String {
        let v = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String
        return v.map { "(\($0))" } ?? ""
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
        
        // Get full PATH from user's shell (macOS app launched from Finder has minimal PATH)
        let shellPath = getShellPath()
        let extendedPath = "/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/local/sbin:" + shellPath
        
        task.environment = [
            "PATH": extendedPath,
            "HOME": NSHomeDirectory(),
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
    
    @objc func showAbout() {
        // Use AppKit's standard About panel — it picks up CFBundleName,
        // CFBundleShortVersionString and CFBundleIdentifier from Info.plist.
        NSApp.activate(ignoringOtherApps: true)
        NSApp.orderFrontStandardAboutPanel(nil)
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
    
    /// Get the user's shell PATH by running their login shell.
    /// macOS apps launched from Finder inherit a minimal PATH.
    private func getShellPath() -> String {
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/bin/sh")
        task.arguments = ["-lc", "echo $PATH"]
        let pipe = Pipe()
        task.standardOutput = pipe
        do {
            try task.run()
            task.waitUntilExit()
            let data = pipe.fileHandleForReading.readDataToEndOfFile()
            if let path = String(data: data, encoding: .utf8) {
                return path.trimmingCharacters(in: .whitespacesAndNewlines)
            }
        } catch {
            print("Failed to get shell PATH: \(error)")
        }
        return "/usr/bin:/bin:/usr/sbin:/sbin"
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
