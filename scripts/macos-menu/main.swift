import Cocoa
import Darwin

class AppDelegate: NSObject, NSApplicationDelegate {
    var statusItem: NSStatusItem!
    var serverTask: Process?
    var serverPort: Int = 3456
    var isRunning = false
    var logFile: FileHandle?
    
    func applicationDidFinishLaunching(_ notification: Notification) {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        statusItem.button?.title = "📡"
        
        let menu = NSMenu()
        
        let titleItem = NSMenuItem(title: "WeChat Radar", action: nil, keyEquivalent: "")
        titleItem.isEnabled = false
        menu.addItem(titleItem)
        menu.addItem(NSMenuItem.separator())
        
        let openItem = NSMenuItem(title: "Open Dashboard", action: #selector(openDashboard), keyEquivalent: "o")
        menu.addItem(openItem)
        
        let toggleItem = NSMenuItem(title: "Start Server", action: #selector(toggleServer), keyEquivalent: "s")
        toggleItem.tag = 100
        menu.addItem(toggleItem)
        
        menu.addItem(NSMenuItem.separator())
        
        let quitItem = NSMenuItem(title: "Quit", action: #selector(quitApp), keyEquivalent: "q")
        menu.addItem(quitItem)
        
        statusItem.menu = menu
        
        startServer()
    }
    
    @objc func openDashboard() {
        let url = URL(string: "http://localhost:\(serverPort)")!
        NSWorkspace.shared.open(url)
    }
    
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
        // Use WeChatRadarServer instead of node for better process naming in Activity Monitor
        let nodePath = resourcesPath + "/node/bin/WeChatRadarServer"
        let appPath = resourcesPath + "/app/server.js"
        let appDir = resourcesPath + "/app"
        
        let fm = FileManager.default
        guard fm.fileExists(atPath: nodePath) else {
            // Fallback to node if renamed binary doesn't exist
            let fallbackPath = resourcesPath + "/node/bin/node"
            guard fm.fileExists(atPath: fallbackPath) else {
                showAlert(message: "Node.js not found in app bundle")
                return
            }
            startServerWithNode(path: fallbackPath, appPath: appPath, appDir: appDir)
            return
        }
        
        startServerWithNode(path: nodePath, appPath: appPath, appDir: appDir)
    }
    
    func startServerWithNode(path: String, appPath: String, appDir: String) {
        serverPort = findAvailablePort(startingAt: 3456)
        
        let task = Process()
        task.executableURL = URL(fileURLWithPath: path)
        task.arguments = [appPath]
        task.environment = [
            "PORT": "\(serverPort)",
            "WECHAT_RADAR_DATA_DIR": NSHomeDirectory() + "/.wechat-radar",
            "NODE_ENV": "production"
        ]
        task.currentDirectoryURL = URL(fileURLWithPath: appDir)
        
        let logPath = NSHomeDirectory() + "/.wechat-radar/server.log"
        if FileManager.default.fileExists(atPath: logPath) {
            try? FileManager.default.removeItem(atPath: logPath)
        }
        FileManager.default.createFile(atPath: logPath, contents: nil, attributes: nil)
        if let logHandle = FileHandle(forWritingAtPath: logPath) {
            task.standardOutput = logHandle
            task.standardError = logHandle
            logFile = logHandle
        }
        
        do {
            try task.run()
            serverTask = task
            isRunning = true
            statusItem.button?.title = "📡"
            updateMenu()
            
            DispatchQueue.global().async {
                for _ in 0..<30 {
                    if self.isServerReady() {
                        DispatchQueue.main.async {
                            self.openDashboard()
                        }
                        break
                    }
                    Thread.sleep(forTimeInterval: 1)
                }
            }
        } catch {
            showAlert(message: "Failed to start server: \(error.localizedDescription)")
        }
    }
    
    func stopServer() {
        guard let task = serverTask else { return }
        task.terminate()
        serverTask = nil
        isRunning = false
        statusItem.button?.title = "📡"
        updateMenu()
        logFile?.closeFile()
        logFile = nil
    }
    
    func isServerReady() -> Bool {
        guard let url = URL(string: "http://localhost:\(serverPort)/api/setup") else { return false }
        var request = URLRequest(url: url)
        request.timeoutInterval = 2
        let semaphore = DispatchSemaphore(value: 0)
        var ready = false
        let task = URLSession.shared.dataTask(with: request) { _, response, _ in
            ready = (response as? HTTPURLResponse)?.statusCode == 200
            semaphore.signal()
        }
        task.resume()
        semaphore.wait()
        return ready
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
    
    func updateMenu() {
        guard let menu = statusItem.menu else { return }
        if let item = menu.item(withTag: 100) {
            item.title = isRunning ? "Stop Server" : "Start Server"
        }
    }
    
    func showAlert(message: String) {
        let alert = NSAlert()
        alert.messageText = message
        alert.alertStyle = .critical
        alert.runModal()
    }
    
    @objc func quitApp() {
        stopServer()
        NSApplication.shared.terminate(nil)
    }
    
    func applicationWillTerminate(_ notification: Notification) {
        stopServer()
    }
}

let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.run()
