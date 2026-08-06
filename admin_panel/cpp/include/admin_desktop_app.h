/**
 * TigerWallet Admin Desktop Application Header
 * Complete C++ Implementation with Qt
 * High Performance, Ultra Low Latency
 */

#ifndef ADMIN_DESKTOP_APP_H
#define ADMIN_DESKTOP_APP_H

#include <QMainWindow>
#include <QWidget>
#include <QPushButton>
#include <QLabel>
#include <QLineEdit>
#include <QTextEdit>
#include <QTableWidget>
#include <QStackedWidget>
#include <QComboBox>
#include <QCheckBox>
#include <QRadioButton>
#include <QToolBar>
#include <QGroupBox>
#include <QFormLayout>
#include <QVBoxLayout>
#include <QHBoxLayout>
#include <QGridLayout>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QString>
#include <QList>
#include <QMessageBox>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>

/**
 * Admin Desktop Application Class
 * Complete desktop application for TigerWallet Admin Panel
 */
class AdminDesktopApp : public QMainWindow {
    Q_OBJECT

public:
    /**
     * Constructor
     * @param parent Parent widget
     */
    explicit AdminDesktopApp(QWidget *parent = nullptr);
    
    /**
     * Destructor
     */
    ~AdminDesktopApp();

private slots:
    /**
     * Navigation item clicked handler
     */
    void onNavItemClicked();
    
    /**
     * Logout button handler
     */
    void onLogout();
    
    /**
     * Toggle between light and dark theme
     */
    void toggleTheme();
    
    /**
     * Network reply received handler
     * @param reply Network reply
     */
    void onNetworkReply(QNetworkReply *reply);

private:
    // UI Components
    QWidget *centralWidget;
    QHBoxLayout *mainLayout;
    
    // Sidebar
    QWidget *sidebarWidget;
    QVBoxLayout *sidebarLayout;
    QList<QPushButton *> navButtons;
    
    // Content Area
    QWidget *contentWidget;
    QVBoxLayout *contentLayout;
    
    // Header
    QWidget *headerWidget;
    QHBoxLayout *headerLayout;
    QLabel *pageTitle;
    QLineEdit *searchBox;
    QPushButton *themeToggle;
    QPushButton *userProfileBtn;
    
    // Content Stack
    QStackedWidget *contentStack;
    
    // Toolbar
    QToolBar *toolbar;
    
    // Tables
    QTableWidget *usersTable;
    QTableWidget *kycTable;
    QTableWidget *transactionsTable;
    QTableWidget *withdrawalsTable;
    QTableWidget *tokensTable;
    QTableWidget *feesTable;
    QTableWidget *botsTable;
    QTableWidget *ticketsTable;
    
    // Network
    QNetworkAccessManager *networkManager;
    QString baseUrl;
    QString authToken;
    QString currentRequestType;
    
    // Theme
    bool isDarkMode;
    
    // UI Setup Methods
    void setupUI();
    void createSidebar();
    void createContentArea();
    void createToolbar();
    void setupNetworkManager();
    void setupTheme();
    
    // Page Creation Methods
    void createDashboardPage();
    void createUsersPage();
    void createKYCPage();
    void createTransactionsPage();
    void createWithdrawalsPage();
    void createTokensPage();
    void createFeesPage();
    void createBotsPage();
    void createAnalyticsPage();
    void createSupportPage();
    void createNotificationsPage();
    void createSettingsPage();
    
    // Theme Methods
    void applyDarkTheme();
    void applyLightTheme();
    
    // Helper Methods
    QWidget* createStatCard(const QString &title, const QString &value, const QString &color);
    
    // Data Loading Methods
    void loadDashboard();
    void loadUsers();
    void loadKYC();
    void loadTransactions();
    void loadWithdrawals();
    void loadTokens();
    void loadFees();
    void loadBots();
    
    // Network Methods
    void makeAuthenticatedRequest(const QString &endpoint);
    
    // Response Parsing Methods
    void parseDashboardResponse(const QJsonObject &obj);
    void parseUsersResponse(const QJsonObject &obj);
    void parseKYCResponse(const QJsonObject &obj);
    void parseTransactionsResponse(const QJsonObject &obj);
    void parseWithdrawalsResponse(const QJsonObject &obj);
    void parseTokensResponse(const QJsonObject &obj);
    void parseFeesResponse(const QJsonObject &obj);
    void parseBotsResponse(const QJsonObject &obj);
};

#endif // ADMIN_DESKTOP_APP_H
