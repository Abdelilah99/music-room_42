import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_sign_in/google_sign_in.dart';

// serverClientId must equal the server's GOOGLE_CLIENT_ID so the returned
// idToken.aud passes server-side verification (see server/internal/auth/google.go).
class GoogleSignInService {
  // initialize() must be awaited before any other call; storing the future
  // lets the constructor remain synchronous while still being awaitable.
  late final Future<void> _ready;

  GoogleSignInService() {
    final clientId = dotenv.env['GOOGLE_CLIENT_ID'];
    _ready = GoogleSignIn.instance.initialize(
      serverClientId: (clientId != null && clientId.isNotEmpty) ? clientId : null,
    );
  }

  // Returns the Google ID token, or null when the user cancels.
  Future<String?> getIdToken() async {
    await _ready;
    try {
      final account = await GoogleSignIn.instance.authenticate();
      return account.authentication.idToken;
    } on GoogleSignInException catch (e) {
      if (e.code == GoogleSignInExceptionCode.canceled) return null;
      rethrow;
    }
  }

  // Signs out first to force a fresh authentication prompt.
  // Use for account linking to avoid returning a cached/stale token.
  Future<String?> getFreshIdToken() async {
    await _ready;
    await GoogleSignIn.instance.signOut();
    return getIdToken();
  }
}

final googleSignInServiceProvider = Provider<GoogleSignInService>(
  (_) => GoogleSignInService(),
);
