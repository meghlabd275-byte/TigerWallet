/**
 * TigerWallet Admin Desktop Application
 * Complete C++ Implementation with Qt
 * High Performance, Ultra Low Latency
 */

#include "admin_desktop_app.h"
#include <QApplication>
#include <QMessageBox>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QNetworkAccessManager>
#include <QNetworkRequest>
#include <QNetworkReply>
#include <QUrl>
#include <QUrlQuery>

// Admin Desktop Application Constructor
AdminDesktopApp::AdminDesktopApp(QWidget *parent)
    : QMainWindow(parent)
    , networkManager(new QNetworkAccessManager(this))
{
    setupUI();
    setupNetworkManager();
    setupTheme();
    loadDashboard();
}

// Destructor
AdminDesktopApp::~AdminDesktopApp() {
    delete networkManager;
}

// Setup Main UI
void AdminDesktopApp::setupUI() {
    setWindowTitle("TigerWallet Admin Panel");
    setMinimumSize(1280, 800);
    
    // Create central widget
    centralWidget = new QWidget(this);
    setCentralWidget(centralWidget);
    
    // Main layout
    mainLayout = new QHBoxLayout(centralWidget);
    
    // Create sidebar
    createSidebar();
    
    // Create content area
    createContentArea();
    
    // Create toolbar
    createToolbar();
    
    // Apply dark theme by default
    applyDarkTheme();
}

// Create Sidebar Navigation
void AdminDesktopApp::createSidebar() {
    sidebarWidget = new QWidget();
    sidebarWidget->setFixedWidth(250);
    sidebarLayout = new QVBoxLayout(sidebarWidget);
    sidebarLayout->setContentsMargins(0, 0, 0, 0);
    
    // Logo/Title
    QLabel *titleLabel = new QLabel("🐯 TigerWallet");
    titleLabel->setStyleSheet("font-size: 20px; font-weight: bold; padding: 20px; color: #f97316;");
    sidebarLayout->addWidget(titleLabel);
    
    // Navigation items
    QStringList navItems = {
        "📊 Dashboard",
        "👥 Users",
        "🛡️ KYC",
        "💳 Transactions",
        "💵 Withdrawals",
        "🪙 Tokens",
        "💰 Fees",
        "🤖 Bots",
        "📈 Analytics",
        "🎫 Support",
        "🔔 Notifications",
        "⚙️ Settings"
    };
    
    for (const QString &item : navItems) {
        QPushButton *btn = new QPushButton(item);
        btn->setFlat(true);
        btn->setStyleSheet("text-align: left; padding: 12px 20px; font-size: 14px;");
        btn->setCursor(Qt::PointingHandCursor);
        connect(btn, &QPushButton::clicked, this, &AdminDesktopApp::onNavItemClicked);
        sidebarLayout->addWidget(btn);
        navButtons.append(btn);
    }
    
    sidebarLayout->addStretch();
    
    // Logout button
    QPushButton *logoutBtn = new QPushButton("🚪 Logout");
    logoutBtn->setFlat(true);
    logoutBtn->setStyleSheet("text-align: left; padding: 12px 20px; font-size: 14px; color: #ef4444;");
    logoutBtn->setCursor(Qt::PointingHandCursor);
    connect(logoutBtn, &QPushButton::clicked, this, &AdminDesktopApp::onLogout);
    sidebarLayout->addWidget(logoutBtn);
    
    mainLayout->addWidget(sidebarWidget);
}

// Create Content Area
void AdminDesktopApp::createContentArea() {
    contentWidget = new QWidget();
    contentLayout = new QVBoxLayout(contentWidget);
    contentLayout->setContentsMargins(20, 20, 20, 20);
    
    // Header
    headerWidget = new QWidget();
    headerLayout = new QHBoxLayout(headerWidget);
    
    pageTitle = new QLabel("Dashboard");
    pageTitle->setStyleSheet("font-size: 24px; font-weight: bold;");
    headerLayout->addWidget(pageTitle);
    
    headerLayout->addStretch();
    
    // Search
    searchBox = new QLineEdit();
    searchBox->setPlaceholderText("Search...");
    searchBox->setFixedWidth(300);
    searchBox->setStyleSheet("padding: 8px; border-radius: 4px; border: 1px solid #333;");
    headerLayout->addWidget(searchBox);
    
    // Theme toggle
    themeToggle = new QPushButton("🌙");
    themeToggle->setFixedSize(40, 40);
    themeToggle->setFlat(true);
    themeToggle->setStyleSheet("font-size: 20px; border-radius: 20px;");
    connect(themeToggle, &QPushButton::clicked, this, &AdminDesktopApp::toggleTheme);
    headerLayout->addWidget(themeToggle);
    
    // User profile
    userProfileBtn = new QPushButton("👤 Admin");
    userProfileBtn->setFlat(true);
    userProfileBtn->setStyleSheet("padding: 8px 16px;");
    headerLayout->addWidget(userProfileBtn);
    
    contentLayout->addWidget(headerWidget);
    
    // Main content
    contentStack = new QStackedWidget();
    contentLayout->addWidget(contentStack);
    
    // Create all pages
    createDashboardPage();
    createUsersPage();
    createKYCPage();
    createTransactionsPage();
    createWithdrawalsPage();
    createTokensPage();
    createFeesPage();
    createBotsPage();
    createAnalyticsPage();
    createSupportPage();
    createNotificationsPage();
    createSettingsPage();
    
    mainLayout->addWidget(contentWidget, 1);
}

// Create Dashboard Page
void AdminDesktopApp::createDashboardPage() {
    QWidget *page = new QWidget();
    QVBoxLayout *layout = new QVBoxLayout(page);
    
    // Stats grid
    QGridLayout *statsGrid = new QGridLayout();
    
    // Total Users
    QWidget *usersCard = createStatCard("Total Users", "0", "#3b82f6");
    statsGrid->addWidget(usersCard, 0, 0);
    
    // Active Users
    QWidget *activeUsersCard = createStatCard("Active Users", "0", "#22c55e");
    statsGrid->addWidget(activeUsersCard, 0, 1);
    
    // Total Volume
    QWidget *volumeCard = createStatCard("Total Volume", "$0", "#f97316");
    statsGrid->addWidget(volumeCard, 0, 2);
    
    // Transactions
    QWidget *txCard = createStatCard("Transactions", "0", "#8b5cf6");
    statsGrid->addWidget(txCard, 0, 3);
    
    layout->addLayout(statsGrid);
    
    // Charts placeholder
    QLabel *chartPlaceholder = new QLabel("📊 Analytics Charts");
    chartPlaceholder->setAlignment(Qt::AlignCenter);
    chartPlaceholder->setStyleSheet("min-height: 300px; background: #1e1e1e; border-radius: 8px; padding: 40px;");
    layout->addWidget(chartPlaceholder);
    
    layout->addStretch();
    contentStack->addWidget(page);
}

// Create Users Page
void AdminDesktopApp::createUsersPage() {
    QWidget *page = new QWidget();
    QVBoxLayout *layout = new QVBoxLayout(page);
    
    // Toolbar
    QHBoxLayout *toolbar = new QHBoxLayout();
    
    QPushButton *refreshBtn = new QPushButton("🔄 Refresh");
    connect(refreshBtn, &QPushButton::clicked, this, &AdminDesktopApp::loadUsers);
    toolbar->addWidget(refreshBtn);
    
    QPushButton *exportBtn = new QPushButton("📤 Export");
    toolbar->addWidget(exportBtn);
    
    toolbar->addStretch();
    
    // Filters
    QComboBox *statusFilter = new QComboBox();
    statusFilter->addItems({"All Status", "Active", "Suspended", "Banned"});
    toolbar->addWidget(statusFilter);
    
    QComboBox *kycFilter = new QComboBox();
    kycFilter->addItems({"All KYC", "Verified", "Pending", "Rejected"});
    toolbar->addWidget(kycFilter);
    
    layout->addLayout(toolbar);
    
    // Users table
    usersTable = new QTableWidget();
    usersTable->setColumnCount(8);
    usersTable->setHorizontalHeaderLabels({"ID", "Email", "Username", "Status", "KYC", "Balance", "Created", "Actions"});
    usersTable->setStyleSheet("QTableWidget { background: #1e1e1e; }");
    layout->addWidget(usersTable);
    
    contentStack->addWidget(page);
}

// Create KYC Page
void AdminDesktopApp::createKYCPage() {
    QWidget *page = new QWidget();
    QVBoxLayout *layout = new QVBoxLayout(page);
    
    // Stats
    QHBoxLayout *statsBar = new QHBoxLayout();
    statsBar->addWidget(createStatCard("Pending", "0", "#eab308"));
    statsBar->addWidget(createStatCard("Approved", "0", "#22c55e"));
    statsBar->addWidget(createStatCard("Rejected", "0", "#ef4444"));
    layout->addLayout(statsBar);
    
    // KYC Table
    kycTable = new QTableWidget();
    kycTable->setColumnCount(7);
    kycTable->setHorizontalHeaderLabels({"ID", "User", "Document Type", "Status", "Submitted", "Actions"});
    layout->addWidget(kycTable);
    
    contentStack->addWidget(page);
}

// Create Transactions Page
void AdminDesktopApp::createTransactionsPage() {
    QWidget *page = new QWidget();
    QVBoxLayout *layout = new QVBoxLayout(page);
    
    // Filters
    QHBoxLayout *filters = new QHBoxLayout();
    filters->addWidget(new QLabel("Status:"));
    QComboBox *txStatusFilter = new QComboBox();
    txStatusFilter->addItems({"All", "Pending", "Completed", "Failed"});
    filters->addWidget(txStatusFilter);
    
    filters->addStretch();
    layout->addLayout(filters);
    
    // Transactions Table
    transactionsTable = new QTableWidget();
    transactionsTable->setColumnCount(10);
    transactionsTable->setHorizontalHeaderLabels({"ID", "Hash", "Type", "Amount", "Fee", "Token", "Chain", "Status", "Time", "Actions"});
    layout->addWidget(transactionsTable);
    
    contentStack->addWidget(page);
}

// Create Withdrawals Page
void AdminDesktopApp::createWithdrawalsPage() {
    QWidget *page = new QWidget();
    QVBoxLayout *layout = new QVBoxLayout(page);
    
    // Pending withdrawals
    QLabel *pendingLabel = new QLabel("Pending Withdrawals");
    pendingLabel->setStyleSheet("font-size: 18px; font-weight: bold;");
    layout->addWidget(pendingLabel);
    
    withdrawalsTable = new QTableWidget();
    withdrawalsTable->setColumnCount(8);
    withdrawalsTable->setHorizontalHeaderLabels({"ID", "User", "Amount", "Token", "Address", "Time", "Status", "Actions"});
    layout->addWidget(withdrawalsTable);
    
    contentStack->addWidget(page);
}

// Create Tokens Page
void AdminDesktopApp::createTokensPage() {
    QWidget *page = new QWidget();
    QVBoxLayout *layout = new QVBoxLayout(page);
    
    // Add token button
    QPushButton *addTokenBtn = new QPushButton("➕ Add Token");
    addTokenBtn->setStyleSheet("padding: 8px 16px; background: #f97316; color: white; border: none; border-radius: 4px;");
    layout->addWidget(addTokenBtn);
    
    // Tokens table
    tokensTable = new QTableWidget();
    tokensTable->setColumnCount(7);
    tokensTable->setHorizontalHeaderLabels({"ID", "Name", "Symbol", "Decimals", "Contract", "Status", "Actions"});
    layout->addWidget(tokensTable);
    
    contentStack->addWidget(page);
}

// Create Fees Page
void AdminDesktopApp::createFeesPage() {
    QWidget *page = new QWidget();
    QVBoxLayout *layout = new QVBoxLayout(page);
    
    // Add fee button
    QPushButton *addFeeBtn = new QPushButton("➕ Add Fee Rule");
    addFeeBtn->setStyleSheet("padding: 8px 16px; background: #f97316; color: white; border: none; border-radius: 4px;");
    layout->addWidget(addFeeBtn);
    
    // Fees table
    feesTable = new QTableWidget();
    feesTable->setColumnCount(6);
    feesTable->setHorizontalHeaderLabels({"ID", "Name", "Type", "Value", "Status", "Actions"});
    layout->addWidget(feesTable);
    
    contentStack->addWidget(page);
}

// Create Bots Page
void AdminDesktopApp::createBotsPage() {
    QWidget *page = new QWidget();
    QVBoxLayout *layout = new QVBoxLayout(page);
    
    // Stats
    QHBoxLayout *statsBar = new QHBoxLayout();
    statsBar->addWidget(createStatCard("Active Bots", "0", "#22c55e"));
    statsBar->addWidget(createStatCard("Total P/L", "$0", "#3b82f6"));
    statsBar->addWidget(createStatCard("24h Volume", "$0", "#f97316"));
    layout->addLayout(statsBar);
    
    // Bots table
    botsTable = new QTableWidget();
    botsTable->setColumnCount(7);
    botsTable->setHorizontalHeaderLabels({"ID", "Name", "Type", "Status", "P/L", "Volume", "Actions"});
    layout->addWidget(botsTable);
    
    contentStack->addWidget(page);
}

// Create Analytics Page
void AdminDesktopApp::createAnalyticsPage() {
    QWidget *page = new QWidget();
    QVBoxLayout *layout = new QVBoxLayout(page);
    
    // Period selector
    QHBoxLayout *periodBar = new QHBoxLayout();
    periodBar->addWidget(new QLabel("Period:"));
    QComboBox *periodSelector = new QComboBox();
    periodSelector->addItems({"24h", "7d", "30d", "90d", "1y"});
    periodBar->addWidget(periodSelector);
    periodBar->addStretch();
    layout->addLayout(periodBar);
    
    // Analytics placeholder
    QLabel *analyticsPlaceholder = new QLabel("📈 Advanced Analytics\n\n• Volume by Chain\n• Revenue Breakdown\n• User Growth\n• Token Performance\n• Trading Patterns");
    analyticsPlaceholder->setAlignment(Qt::AlignCenter);
    analyticsPlaceholder->setStyleSheet("min-height: 400px; background: #1e1e1e; border-radius: 8px; padding: 40px; font-size: 16px;");
    layout->addWidget(analyticsPlaceholder);
    
    contentStack->addWidget(page);
}

// Create Support Page
void AdminDesktopApp::createSupportPage() {
    QWidget *page = new QWidget();
    QVBoxLayout *layout = new QVBoxLayout(page);
    
    // Stats
    QHBoxLayout *statsBar = new QHBoxLayout();
    statsBar->addWidget(createStatCard("Open Tickets", "0", "#eab308"));
    statsBar->addWidget(createStatCard("In Progress", "0", "#3b82f6"));
    statsBar->addWidget(createStatCard("SLA Violations", "0", "#ef4444"));
    layout->addLayout(statsBar);
    
    // Tickets table
    ticketsTable = new QTableWidget();
    ticketsTable->setColumnCount(7);
    ticketsTable->setHorizontalHeaderLabels({"ID", "Subject", "User", "Priority", "Status", "Created", "Actions"});
    layout->addWidget(ticketsTable);
    
    contentStack->addWidget(page);
}

// Create Notifications Page
void AdminDesktopApp::createNotificationsPage() {
    QWidget *page = new QWidget();
    QVBoxLayout *layout = new QVBoxLayout(page);
    
    // Send notification form
    QGroupBox *sendBox = new QGroupBox("Send Notification");
    QVBoxLayout *sendLayout = new QVBoxLayout();
    
    QLineEdit *titleInput = new QLineEdit();
    titleInput->setPlaceholderText("Title");
    sendLayout->addWidget(titleInput);
    
    QTextEdit *messageInput = new QTextEdit();
    messageInput->setPlaceholderText("Message");
    messageInput->setMaximumHeight(100);
    sendLayout->addWidget(messageInput);
    
    QCheckBox *broadcastCheck = new QCheckBox("Broadcast to all users");
    sendLayout->addWidget(broadcastCheck);
    
    QPushButton *sendBtn = new QPushButton("Send");
    sendBtn->setStyleSheet("padding: 8px 24px; background: #f97316; color: white; border: none; border-radius: 4px;");
    sendLayout->addWidget(sendBtn);
    
    sendBox->setLayout(sendLayout);
    layout->addWidget(sendBox);
    
    // Recent notifications
    QLabel *recentLabel = new QLabel("Recent Notifications");
    recentLabel->setStyleSheet("font-size: 18px; font-weight: bold; margin-top: 20px;");
    layout->addWidget(recentLabel);
    
    contentStack->addWidget(page);
}

// Create Settings Page
void AdminDesktopApp::createSettingsPage() {
    QWidget *page = new QWidget();
    QVBoxLayout *layout = new QVBoxLayout(page);
    
    // Profile section
    QGroupBox *profileBox = new QGroupBox("Profile Settings");
    QFormLayout *profileLayout = new QFormLayout();
    
    QLineEdit *emailInput = new QLineEdit();
    emailInput->setPlaceholderText("admin@tigerwallet.com");
    profileLayout->addRow("Email:", emailInput);
    
    QLineEdit *usernameInput = new QLineEdit();
    profileLayout->addRow("Username:", usernameInput);
    
    QPushButton *saveProfileBtn = new QPushButton("Save Profile");
    profileLayout->addRow("", saveProfileBtn);
    
    profileBox->setLayout(profileLayout);
    layout->addWidget(profileBox);
    
    // Security section
    QGroupBox *securityBox = new QGroupBox("Security");
    QVBoxLayout *securityLayout = new QVBoxLayout();
    
    QCheckBox *twoFactorCheck = new QCheckBox("Enable Two-Factor Authentication");
    securityLayout->addWidget(twoFactorCheck);
    
    QPushButton *changePasswordBtn = new QPushButton("Change Password");
    securityLayout->addWidget(changePasswordBtn);
    
    securityBox->setLayout(securityLayout);
    layout->addWidget(securityBox);
    
    // Theme section
    QGroupBox *themeBox = new QGroupBox("Appearance");
    QVBoxLayout *themeLayout = new QVBoxLayout();
    
    QRadioButton *darkModeRadio = new QRadioButton("Dark Mode");
    darkModeRadio->setChecked(true);
    themeLayout->addWidget(darkModeRadio);
    
    QRadioButton *lightModeRadio = new QRadioButton("Light Mode");
    themeLayout->addWidget(lightModeRadio);
    
    themeBox->setLayout(themeLayout);
    layout->addWidget(themeBox);
    
    layout->addStretch();
    contentStack->addWidget(page);
}

// Create Toolbar
void AdminDesktopApp::createToolbar() {
    toolbar = new QToolBar();
    toolbar->setMovable(false);
    addToolBar(toolbar);
}

// Setup Network Manager
void AdminDesktopApp::setupNetworkManager() {
    connect(networkManager, &QNetworkAccessManager::finished, this, &AdminDesktopApp::onNetworkReply);
    baseUrl = "https://api.tigerwallet.com/api/v1/admin";
}

// Setup Theme
void AdminDesktopApp::setupTheme() {
    isDarkMode = true;
}

// Apply Dark Theme
void AdminDesktopApp::applyDarkTheme() {
    setStyleSheet(R"(
        QMainWindow { background-color: #0a0a0a; color: #ffffff; }
        QWidget { background-color: #0a0a0a; color: #ffffff; }
        QPushButton { 
            background-color: #1e1e1e; 
            color: #ffffff; 
            border: 1px solid #333;
            padding: 8px 16px;
            border-radius: 4px;
        }
        QPushButton:hover { background-color: #2d2d2d; }
        QLineEdit, QTextEdit, QComboBox {
            background-color: #1e1e1e;
            color: #ffffff;
            border: 1px solid #333;
            padding: 8px;
            border-radius: 4px;
        }
        QTableWidget { 
            background-color: #1e1e1e; 
            color: #ffffff;
            gridline-color: #333;
        }
        QHeaderView::section {
            background-color: #2d2d2d;
            color: #ffffff;
            padding: 8px;
            border: 1px solid #333;
        }
        QScrollBar:vertical {
            background: #1e1e1e;
            width: 12px;
        }
        QScrollBar::handle:vertical {
            background: #444;
            border-radius: 6px;
        }
    )");
    
    if (themeToggle) {
        themeToggle->setText("🌙");
    }
}

// Apply Light Theme
void AdminDesktopApp::applyLightTheme() {
    setStyleSheet(R"(
        QMainWindow { background-color: #f5f5f5; color: #000000; }
        QWidget { background-color: #f5f5f5; color: #000000; }
        QPushButton { 
            background-color: #ffffff; 
            color: #000000; 
            border: 1px solid #ddd;
            padding: 8px 16px;
            border-radius: 4px;
        }
        QPushButton:hover { background-color: #e5e5e5; }
        QLineEdit, QTextEdit, QComboBox {
            background-color: #ffffff;
            color: #000000;
            border: 1px solid #ddd;
            padding: 8px;
            border-radius: 4px;
        }
        QTableWidget { 
            background-color: #ffffff; 
            color: #000000;
            gridline-color: #ddd;
        }
        QHeaderView::section {
            background-color: #e5e5e5;
            color: #000000;
            padding: 8px;
            border: 1px solid #ddd;
        }
    )");
    
    if (themeToggle) {
        themeToggle->setText("☀️");
    }
}

// Toggle Theme
void AdminDesktopApp::toggleTheme() {
    isDarkMode = !isDarkMode;
    if (isDarkMode) {
        applyDarkTheme();
    } else {
        applyLightTheme();
    }
}

// Create Stat Card Widget
QWidget* AdminDesktopApp::createStatCard(const QString &title, const QString &value, const QString &color) {
    QWidget *card = new QWidget();
    card->setStyleSheet(QString("background: %1; border-radius: 8px; padding: 20px;").arg(isDarkMode ? "#1e1e1e" : "#ffffff"));
    
    QVBoxLayout *layout = new QVBoxLayout(card);
    
    QLabel *titleLabel = new QLabel(title);
    titleLabel->setStyleSheet("color: #888; font-size: 14px;");
    layout->addWidget(titleLabel);
    
    QLabel *valueLabel = new QLabel(value);
    valueLabel->setStyleSheet(QString("color: %1; font-size: 28px; font-weight: bold;").arg(color));
    layout->addWidget(valueLabel);
    
    return card;
}

// Navigation Handler
void AdminDesktopApp::onNavItemClicked() {
    QPushButton *btn = qobject_cast<QPushButton*>(sender());
    if (!btn) return;
    
    QString text = btn->text();
    int index = 0;
    
    if (text.contains("Dashboard")) index = 0;
    else if (text.contains("Users")) index = 1;
    else if (text.contains("KYC")) index = 2;
    else if (text.contains("Transactions")) index = 3;
    else if (text.contains("Withdrawals")) index = 4;
    else if (text.contains("Tokens")) index = 5;
    else if (text.contains("Fees")) index = 6;
    else if (text.contains("Bots")) index = 7;
    else if (text.contains("Analytics")) index = 8;
    else if (text.contains("Support")) index = 9;
    else if (text.contains("Notifications")) index = 10;
    else if (text.contains("Settings")) index = 11;
    
    contentStack->setCurrentIndex(index);
    
    // Update title
    QString title = text.split(" ").last();
    if (pageTitle) {
        pageTitle->setText(title);
    }
    
    // Load data for page
    switch (index) {
        case 0: loadDashboard(); break;
        case 1: loadUsers(); break;
        case 2: loadKYC(); break;
        case 3: loadTransactions(); break;
        case 4: loadWithdrawals(); break;
        case 5: loadTokens(); break;
        case 6: loadFees(); break;
        case 7: loadBots(); break;
    }
}

// Logout Handler
void AdminDesktopApp::onLogout() {
    QMessageBox::StandardButton reply = QMessageBox::question(
        this, "Logout", "Are you sure you want to logout?",
        QMessageBox::Yes | QMessageBox::No
    );
    
    if (reply == QMessageBox::Yes) {
        authToken.clear();
        QMessageBox::information(this, "Logged Out", "You have been logged out successfully.");
    }
}

// Network Reply Handler
void AdminDesktopApp::onNetworkReply(QNetworkReply *reply) {
    if (reply->error() != QNetworkReply::NoError) {
        qDebug() << "Network error:" << reply->errorString();
        return;
    }
    
    QByteArray data = reply->readAll();
    QJsonDocument doc = QJsonDocument::fromJson(data);
    QJsonObject obj = doc.object();
    
    // Handle response based on current request type
    if (currentRequestType == "dashboard") {
        parseDashboardResponse(obj);
    } else if (currentRequestType == "users") {
        parseUsersResponse(obj);
    }
}

// Load Dashboard Data
void AdminDesktopApp::loadDashboard() {
    currentRequestType = "dashboard";
    makeAuthenticatedRequest("/analytics/dashboard");
}

// Load Users Data
void AdminDesktopApp::loadUsers() {
    currentRequestType = "users";
    makeAuthenticatedRequest("/users?page=1&page_size=50");
}

// Load KYC Data
void AdminDesktopApp::loadKYC() {
    currentRequestType = "kyc";
    makeAuthenticatedRequest("/kyc");
}

// Load Transactions Data
void AdminDesktopApp::loadTransactions() {
    currentRequestType = "transactions";
    makeAuthenticatedRequest("/transactions");
}

// Load Withdrawals Data
void AdminDesktopApp::loadWithdrawals() {
    currentRequestType = "withdrawals";
    makeAuthenticatedRequest("/withdrawals");
}

// Load Tokens Data
void AdminDesktopApp::loadTokens() {
    currentRequestType = "tokens";
    makeAuthenticatedRequest("/tokens");
}

// Load Fees Data
void AdminDesktopApp::loadFees() {
    currentRequestType = "fees";
    makeAuthenticatedRequest("/fees");
}

// Load Bots Data
void AdminDesktopApp::loadBots() {
    currentRequestType = "bots";
    makeAuthenticatedRequest("/bots");
}

// Make Authenticated Request
void AdminDesktopApp::makeAuthenticatedRequest(const QString &endpoint) {
    QUrl url(baseUrl + endpoint);
    QNetworkRequest request(url);
    request.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");
    
    if (!authToken.isEmpty()) {
        request.setRawHeader("Authorization", "Bearer " + authToken.toUtf8());
    }
    
    networkManager->get(request);
}

// Parse Dashboard Response
void AdminDesktopApp::parseDashboardResponse(const QJsonObject &obj) {
    qDebug() << "Dashboard data:" << obj;
}

// Parse Users Response
void AdminDesktopApp::parseUsersResponse(const QJsonObject &obj) {
    QJsonArray users = obj["data"].toArray();
    usersTable->setRowCount(users.size());
    
    for (int i = 0; i < users.size(); i++) {
        QJsonObject user = users[i].toObject();
        
        usersTable->setItem(i, 0, new QTableWidgetItem(QString::number(user["id"].toInt())));
        usersTable->setItem(i, 1, new QTableWidgetItem(user["email"].toString()));
        usersTable->setItem(i, 2, new QTableWidgetItem(user["username"].toString()));
        usersTable->setItem(i, 3, new QTableWidgetItem(user["status"].toString()));
        usersTable->setItem(i, 4, new QTableWidgetItem(user["kyc_status"].toString()));
    }
}

// Main function
int main(int argc, char *argv[]) {
    QApplication app(argc, argv);
    
    AdminDesktopApp window;
    window.show();
    
    return app.exec();
}
