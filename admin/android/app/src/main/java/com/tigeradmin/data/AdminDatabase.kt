package com.tigeradmin.data

import android.content.Context
import androidx.room.Database
import androidx.room.Room
import androidx.room.RoomDatabase
import androidx.room.TypeConverters

/**
 * Admin Database
 * Local SQLite database for offline caching
 * Note: In production, this should be migrated to use PostgreSQL with R2DBC or Room with SQLite backup to PostgreSQL
 */
@Database(
    entities = [
        AdminEntity::class,
        UserEntity::class,
        TransactionEntity::class,
        KYCEntity::class,
        TokenEntity::class,
        WithdrawalEntity::class,
        WhiteLabelEntity::class,
        SystemLogEntity::class
    ],
    version = 1,
    exportSchema = false
)
@TypeConverters(Converters::class)
abstract class AdminDatabase : RoomDatabase() {

    abstract fun adminDao(): AdminDao
    abstract fun userDao(): UserDao
    abstract fun transactionDao(): TransactionDao
    abstract fun kycDao(): KYCDao
    abstract fun tokenDao(): TokenDao
    abstract fun withdrawalDao(): WithdrawalDao
    abstract fun whiteLabelDao(): WhiteLabelDao
    abstract fun systemLogDao(): SystemLogDao

    companion object {
        private const val DATABASE_NAME = "tigeradmin_database"

        @Volatile
        private var INSTANCE: AdminDatabase? = null

        fun getInstance(context: Context): AdminDatabase {
            return INSTANCE ?: synchronized(this) {
                val instance = Room.databaseBuilder(
                    context.applicationContext,
                    AdminDatabase::class.java,
                    DATABASE_NAME
                )
                    .fallbackToDestructiveMigration()
                    .build()
                INSTANCE = instance
                instance
            }
        }
    }
}

/**
 * Type Converters for Room
 */
class Converters {
    // Type converters for complex types
}

/**
 * Admin Entity
 */
@Entity(tableName = "admins")
data class AdminEntity(
    @PrimaryKey val id: Long,
    val username: String,
    val email: String,
    val firstName: String?,
    val lastName: String?,
    val role: String,
    val status: String,
    val twoFactorEnabled: Boolean,
    val lastLoginAt: String?,
    val createdAt: String,
    val updatedAt: String,
    val avatarUrl: String?,
    val cachedAt: Long = System.currentTimeMillis()
)

/**
 * User Entity
 */
@Entity(tableName = "users")
data class UserEntity(
    @PrimaryKey val id: Long,
    val email: String,
    val username: String?,
    val walletAddress: String?,
    val status: String,
    val kycStatus: String,
    val kycLevel: Int,
    val riskScore: Int,
    val createdAt: String,
    val lastLoginAt: String?,
    val tags: String, // JSON array as string
    val cachedAt: Long = System.currentTimeMillis()
)

/**
 * Transaction Entity
 */
@Entity(tableName = "transactions")
data class TransactionEntity(
    @PrimaryKey val id: Long,
    val hash: String,
    val type: String,
    val chain: String,
    val fromAddress: String,
    val toAddress: String,
    val amount: String,
    val token: String,
    val status: String,
    val flagged: Boolean,
    val flagReason: String?,
    val userId: Long,
    val timestamp: String,
    val cachedAt: Long = System.currentTimeMillis()
)

/**
 * KYC Entity
 */
@Entity(tableName = "kyc_applications")
data class KYCEntity(
    @PrimaryKey val id: Long,
    val userId: Long,
    val userEmail: String,
    val level: Int,
    val status: String,
    val submittedAt: String,
    val reviewedAt: String?,
    val rejectionReason: String?,
    val cachedAt: Long = System.currentTimeMillis()
)

/**
 * Token Entity
 */
@Entity(tableName = "tokens")
data class TokenEntity(
    @PrimaryKey val id: Long,
    val name: String,
    val symbol: String,
    val contractAddress: String,
    val chain: String,
    val decimals: Int,
    val totalSupply: String,
    val logoUrl: String?,
    val price: String?,
    val isActive: Boolean,
    val isVerified: Boolean,
    val cachedAt: Long = System.currentTimeMillis()
)

/**
 * Withdrawal Entity
 */
@Entity(tableName = "withdrawals")
data class WithdrawalEntity(
    @PrimaryKey val id: Long,
    val userId: Long,
    val userEmail: String,
    val amount: String,
    val token: String,
    val chain: String,
    val toAddress: String,
    val status: String,
    val createdAt: String,
    val cachedAt: Long = System.currentTimeMillis()
)

/**
 * White Label Entity
 */
@Entity(tableName = "white_labels")
data class WhiteLabelEntity(
    @PrimaryKey val id: Long,
    val name: String,
    val slug: String,
    val domain: String?,
    val logoUrl: String?,
    val primaryColor: String,
    val status: String,
    val contactEmail: String?,
    val cachedAt: Long = System.currentTimeMillis()
)

/**
 * System Log Entity
 */
@Entity(tableName = "system_logs")
data class SystemLogEntity(
    @PrimaryKey val id: Long,
    val timestamp: String,
    val level: String,
    val service: String,
    val message: String,
    val cachedAt: Long = System.currentTimeMillis()
)

/**
 * Admin DAO
 */
@Dao
interface AdminDao {
    @Query("SELECT * FROM admins ORDER BY id DESC")
    suspend fun getAllAdmins(): List<AdminEntity>

    @Query("SELECT * FROM admins WHERE id = :id")
    suspend fun getAdminById(id: Long): AdminEntity?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAdmin(admin: AdminEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAdmins(admins: List<AdminEntity>)

    @Delete
    suspend fun deleteAdmin(admin: AdminEntity)

    @Query("DELETE FROM admins")
    suspend fun deleteAllAdmins()

    @Query("DELETE FROM admins WHERE cachedAt < :timestamp")
    suspend fun deleteOldAdmins(timestamp: Long)
}

/**
 * User DAO
 */
@Dao
interface UserDao {
    @Query("SELECT * FROM users ORDER BY id DESC")
    suspend fun getAllUsers(): List<UserEntity>

    @Query("SELECT * FROM users WHERE id = :id")
    suspend fun getUserById(id: Long): UserEntity?

    @Query("SELECT * FROM users WHERE status = :status")
    suspend fun getUsersByStatus(status: String): List<UserEntity>

    @Query("SELECT * FROM users WHERE kycStatus = :kycStatus")
    suspend fun getUsersByKYCStatus(kycStatus: String): List<UserEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertUser(user: UserEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertUsers(users: List<UserEntity>)

    @Delete
    suspend fun deleteUser(user: UserEntity)

    @Query("DELETE FROM users")
    suspend fun deleteAllUsers()
}

/**
 * Transaction DAO
 */
@Dao
interface TransactionDao {
    @Query("SELECT * FROM transactions ORDER BY timestamp DESC")
    suspend fun getAllTransactions(): List<TransactionEntity>

    @Query("SELECT * FROM transactions WHERE id = :id")
    suspend fun getTransactionById(id: Long): TransactionEntity?

    @Query("SELECT * FROM transactions WHERE status = :status")
    suspend fun getTransactionsByStatus(status: String): List<TransactionEntity>

    @Query("SELECT * FROM transactions WHERE flagged = 1")
    suspend fun getFlaggedTransactions(): List<TransactionEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertTransaction(transaction: TransactionEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertTransactions(transactions: List<TransactionEntity>)

    @Delete
    suspend fun deleteTransaction(transaction: TransactionEntity)

    @Query("DELETE FROM transactions")
    suspend fun deleteAllTransactions()
}

/**
 * KYC DAO
 */
@Dao
interface KYCDao {
    @Query("SELECT * FROM kyc_applications ORDER BY submittedAt DESC")
    suspend fun getAllKYCApplications(): List<KYCEntity>

    @Query("SELECT * FROM kyc_applications WHERE id = :id")
    suspend fun getKYCApplicationById(id: Long): KYCEntity?

    @Query("SELECT * FROM kyc_applications WHERE status = :status")
    suspend fun getKYCApplicationsByStatus(status: String): List<KYCEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertKYCApplication(kyc: KYCEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertKYCApplications(kycs: List<KYCEntity>)

    @Delete
    suspend fun deleteKYCApplication(kyc: KYCEntity)

    @Query("DELETE FROM kyc_applications")
    suspend fun deleteAllKYCApplications()
}

/**
 * Token DAO
 */
@Dao
interface TokenDao {
    @Query("SELECT * FROM tokens ORDER BY name ASC")
    suspend fun getAllTokens(): List<TokenEntity>

    @Query("SELECT * FROM tokens WHERE id = :id")
    suspend fun getTokenById(id: Long): TokenEntity?

    @Query("SELECT * FROM tokens WHERE symbol LIKE :symbol")
    suspend fun getTokenBySymbol(symbol: String): TokenEntity?

    @Query("SELECT * FROM tokens WHERE isActive = 1")
    suspend fun getActiveTokens(): List<TokenEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertToken(token: TokenEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertTokens(tokens: List<TokenEntity>)

    @Delete
    suspend fun deleteToken(token: TokenEntity)

    @Query("DELETE FROM tokens")
    suspend fun deleteAllTokens()
}

/**
 * Withdrawal DAO
 */
@Dao
interface WithdrawalDao {
    @Query("SELECT * FROM withdrawals ORDER BY createdAt DESC")
    suspend fun getAllWithdrawals(): List<WithdrawalEntity>

    @Query("SELECT * FROM withdrawals WHERE id = :id")
    suspend fun getWithdrawalById(id: Long): WithdrawalEntity?

    @Query("SELECT * FROM withdrawals WHERE status = :status")
    suspend fun getWithdrawalsByStatus(status: String): List<WithdrawalEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertWithdrawal(withdrawal: WithdrawalEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertWithdrawals(withdrawals: List<WithdrawalEntity>)

    @Delete
    suspend fun deleteWithdrawal(withdrawal: WithdrawalEntity)

    @Query("DELETE FROM withdrawals")
    suspend fun deleteAllWithdrawals()
}

/**
 * White Label DAO
 */
@Dao
interface WhiteLabelDao {
    @Query("SELECT * FROM white_labels ORDER BY name ASC")
    suspend fun getAllWhiteLabels(): List<WhiteLabelEntity>

    @Query("SELECT * FROM white_labels WHERE id = :id")
    suspend fun getWhiteLabelById(id: Long): WhiteLabelEntity?

    @Query("SELECT * FROM white_labels WHERE status = :status")
    suspend fun getWhiteLabelsByStatus(status: String): List<WhiteLabelEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertWhiteLabel(whiteLabel: WhiteLabelEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertWhiteLabels(whiteLabels: List<WhiteLabelEntity>)

    @Delete
    suspend fun deleteWhiteLabel(whiteLabel: WhiteLabelEntity)

    @Query("DELETE FROM white_labels")
    suspend fun deleteAllWhiteLabels()
}

/**
 * System Log DAO
 */
@Dao
interface SystemLogDao {
    @Query("SELECT * FROM system_logs ORDER BY timestamp DESC LIMIT :limit")
    suspend fun getRecentLogs(limit: Int = 100): List<SystemLogEntity>

    @Query("SELECT * FROM system_logs WHERE level = :level ORDER BY timestamp DESC")
    suspend fun getLogsByLevel(level: String): List<SystemLogEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertLog(log: SystemLogEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertLogs(logs: List<SystemLogEntity>)

    @Query("DELETE FROM system_logs")
    suspend fun deleteAllLogs()

    @Query("DELETE FROM system_logs WHERE cachedAt < :timestamp")
    suspend fun deleteOldLogs(timestamp: Long)
}
