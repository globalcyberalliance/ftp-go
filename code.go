package ftp

// FTP status codes.
// See: https://en.wikipedia.org/wiki/List_of_FTP_server_return_codes
const (
	// 100 Series – Positive Preliminary replies.
	CodeRestartMarkerReply        = 110
	CodeServiceReadyInMinutes     = 120
	CodeDataConnectionAlreadyOpen = 125
	CodeFileStatusOK              = 150

	// 200 Series – Positive Completion replies.
	CodeCommandOK                   = 200
	CodeCommandNotImplemented       = 202
	CodeSystemStatus                = 211
	CodeDirectoryStatus             = 212
	CodeFileStatus                  = 213
	CodeHelpMessage                 = 214
	CodeSystemType                  = 215
	CodeServiceReady                = 220
	CodeServiceClosingControlConn   = 221
	CodeDataConnectionOpen          = 225
	CodeClosingDataConnection       = 226
	CodeEnteringPassiveMode         = 227
	CodeEnteringLongPassiveMode     = 228
	CodeEnteringExtendedPassiveMode = 229
	CodeUserLoggedIn                = 230
	CodeUserLoggedInAuthorized      = 232
	CodeSecurityMechanismAccepted   = 234
	CodeSecurityDataAccepted        = 235
	CodeFileActionOK                = 250
	CodeFileAttributesChanged       = 253
	CodePathnameCreated             = 257

	// 300 Series – Positive Intermediate replies.
	CodeUserNameOKNeedPassword      = 331
	CodeNoNeedAccountForLogin       = 332
	CodeSecurityMechanismExchange   = 334
	CodeUsernamePasswordOKChallenge = 336
	CodeRequestedFileActionPending  = 350

	// 400 Series – Transient Negative Completion replies.
	CodeServiceNotAvailable       = 421
	CodeCannotOpenDataConnection  = 425
	CodeConnectionClosed          = 426
	CodeInvalidUsernameOrPassword = 430
	CodeNeedResourceForSecurity   = 431
	CodeRequestedHostUnavailable  = 434
	CodeFileActionNotTaken        = 450
	CodeActionAbortedLocalError   = 451
	CodeInsufficientStorageSpace  = 452

	// 500 Series – Permanent Negative Completion replies.
	CodeSyntaxError                  = 500
	CodeSyntaxErrorInParameters      = 501
	CodeCommandNotImplemented502     = 502
	CodeBadSequenceOfCommands        = 503
	CodeCommandNotImplementedForParm = 504
	CodeNetworkProtocolNotSupported  = 522
	CodeNotLoggedIn                  = 530
	CodeNeedAccountForStoringFiles   = 532
	CodeProtectionLevelDenied        = 533
	CodeRequestDeniedPolicy          = 534
	CodeFailedSecurityCheck          = 535
	CodeDataProtectionNotSupported   = 536
	CodeCommandProtLevelUnsupported  = 537
	CodeFileUnavailable              = 550
	CodeActionAbortedPageUnknown     = 551
	CodeExceededStorageAllocation    = 552
	CodeFileNameNotAllowed           = 553

	// 600 Series – Protected replies.
	CodeIntegrityProtectedReply      = 631
	CodeConfidentialityIntegrityProt = 632
	CodeConfidentialityProtected     = 633
)

// CodeText returns a descriptive text for the FTP status code.
// Returns an empty string if the code is unknown.
func CodeText(code int) string {
	switch code {
	// 100 Series.
	case CodeRestartMarkerReply:
		return "Restart marker reply"
	case CodeServiceReadyInMinutes:
		return "Service ready in nnn minutes"
	case CodeDataConnectionAlreadyOpen:
		return "Data connection already open; transfer starting"
	case CodeFileStatusOK:
		return "File status okay; about to open data connection"

	// 200 Series.
	case CodeCommandOK:
		return "Command okay"
	case CodeCommandNotImplemented:
		return "Command not implemented, superfluous at this site"
	case CodeSystemStatus:
		return "System status, or system help reply"
	case CodeDirectoryStatus:
		return "Directory status"
	case CodeFileStatus:
		return "File status"
	case CodeHelpMessage:
		return "Help message"
	case CodeSystemType:
		return "NAME system type"
	case CodeServiceReady:
		return "Service ready for new user"
	case CodeServiceClosingControlConn:
		return "Service closing control connection"
	case CodeDataConnectionOpen:
		return "Data connection open; no transfer in progress"
	case CodeClosingDataConnection:
		return "Closing data connection. Requested file action successful"
	case CodeEnteringPassiveMode:
		return "Entering Passive Mode"
	case CodeEnteringLongPassiveMode:
		return "Entering Long Passive Mode"
	case CodeEnteringExtendedPassiveMode:
		return "Entering Extended Passive Mode"
	case CodeUserLoggedIn:
		return "User logged in, proceed"
	case CodeUserLoggedInAuthorized:
		return "User logged in, authorized by security data exchange"
	case CodeSecurityMechanismAccepted:
		return "Server accepts the security mechanism; no data needs to be exchanged"
	case CodeSecurityDataAccepted:
		return "Server accepts security data; no further exchange needed"
	case CodeFileActionOK:
		return "Requested file action okay, completed"
	case CodeFileAttributesChanged:
		return "File attributes/date-time changed okay"
	case CodePathnameCreated:
		return "\"PATHNAME\" created"

	// 300 Series.
	case CodeUserNameOKNeedPassword:
		return "User name okay, need password"
	case CodeNoNeedAccountForLogin:
		return "No need account for login"
	case CodeSecurityMechanismExchange:
		return "Security mechanism accepted; some data needs to be exchanged"
	case CodeUsernamePasswordOKChallenge:
		return "Username okay, password okay; challenge issued"
	case CodeRequestedFileActionPending:
		return "Requested file action pending further information"

	// 400 Series.
	case CodeServiceNotAvailable:
		return "Service not available, closing control connection"
	case CodeCannotOpenDataConnection:
		return "Can't open data connection"
	case CodeConnectionClosed:
		return "Connection closed; transfer aborted"
	case CodeInvalidUsernameOrPassword:
		return "Invalid username or password"
	case CodeNeedResourceForSecurity:
		return "Need some unavailable resource to process security"
	case CodeRequestedHostUnavailable:
		return "Requested host unavailable"
	case CodeFileActionNotTaken:
		return "Requested file action not taken"
	case CodeActionAbortedLocalError:
		return "Requested action aborted. Local error in processing"
	case CodeInsufficientStorageSpace:
		return "Insufficient storage space in system. File unavailable"

	// 500 Series.
	case CodeSyntaxError:
		return "Syntax error, command unrecognized"
	case CodeSyntaxErrorInParameters:
		return "Syntax error in parameters or arguments"
	case CodeCommandNotImplemented502:
		return "Command not implemented"
	case CodeBadSequenceOfCommands:
		return "Bad sequence of commands"
	case CodeCommandNotImplementedForParm:
		return "Command not implemented for that parameter"
	case CodeNetworkProtocolNotSupported:
		return "Network protocol not supported, use (list)"
	case CodeNotLoggedIn:
		return "Not logged in"
	case CodeNeedAccountForStoringFiles:
		return "Need account for storing files"
	case CodeProtectionLevelDenied:
		return "Command protection level denied for policy reasons"
	case CodeRequestDeniedPolicy:
		return "Request denied for policy reasons"
	case CodeFailedSecurityCheck:
		return "Failed security check"
	case CodeDataProtectionNotSupported:
		return "Data protection level not supported by security mechanism"
	case CodeCommandProtLevelUnsupported:
		return "Command protection level not supported by security mechanism"
	case CodeFileUnavailable:
		return "Requested action not taken. File unavailable"
	case CodeActionAbortedPageUnknown:
		return "Requested action aborted. Page type unknown"
	case CodeExceededStorageAllocation:
		return "Requested file action aborted. Exceeded storage allocation"
	case CodeFileNameNotAllowed:
		return "Requested action not taken. File name not allowed"

	// 600 Series.
	case CodeIntegrityProtectedReply:
		return "Integrity protected reply"
	case CodeConfidentialityIntegrityProt:
		return "Confidentiality and integrity protected reply"
	case CodeConfidentialityProtected:
		return "Confidentiality protected reply"

	default:
		return ""
	}
}
