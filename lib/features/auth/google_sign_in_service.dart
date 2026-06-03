import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_sign_in/google_sign_in.dart';

// serverClientId must equal the server's GOOGLE_CLIENT_ID so the returned
// idToken.aud passes server-side verification (see server/internal/auth/google.go).
class GoogleSignInService {
  late final GoogleSignIn _googleSignIn;

  GoogleSignInService() {
    final clientId = dotenv.env['GOOGLE_CLIENT_ID'];
    _googleSignIn = GoogleSignIn(
      serverClientId: (clientId != null && clientId.isNotEmpty) ? clientId : null,
    );
  }

  // Returns the Google ID token, or null when the user cancels.
  Future<String?> getIdToken() async {
    final account = await _googleSignIn.signIn();
    if (account == null) return null;
    final auth = await account.authentication;
    return auth.idToken;
  }

  // Signs out first to force a fresh authentication prompt.
  // Use for account linking to avoid returning a cached/stale token.
  Future<String?> getFreshIdToken() async {
    await _googleSignIn.signOut();
    return getIdToken();
  }
}

final googleSignInServiceProvider = Provider<GoogleSignInService>(
  (_) => GoogleSignInService(),
);
