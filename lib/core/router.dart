import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:music_room/features/auth/auth_provider.dart';
import 'package:music_room/features/auth/forgot_password_screen.dart';
import 'package:music_room/features/auth/login_screen.dart';
import 'package:music_room/features/auth/register_screen.dart';
import 'package:music_room/features/auth/verify_notice_screen.dart';
import 'package:music_room/core/models/device.dart';
import 'package:music_room/features/delegation/delegation_screen.dart';
import 'package:music_room/features/devices/device_detail_screen.dart';
import 'package:music_room/features/devices/my_devices_screen.dart';
import 'package:music_room/features/playlist_editor/create_playlist_screen.dart';
import 'package:music_room/features/playlist_editor/playlist_editor_screen.dart';
import 'package:music_room/features/playlist_editor/playlist_list_screen.dart';
import 'package:music_room/features/profile/friends_screen.dart';
import 'package:music_room/features/profile/profile_screen.dart';
import 'package:music_room/features/profile/user_profile_screen.dart';
import 'package:music_room/features/track_vote/create_event_screen.dart';
import 'package:music_room/features/track_vote/event_detail_screen.dart';
import 'package:music_room/features/track_vote/event_list_screen.dart';

// ── Router provider ──────────────────────────────────────────────────────────

final routerProvider = Provider<GoRouter>((ref) {
  final notifier = _RouterNotifier(ref);
  ref.onDispose(notifier.dispose);

  return GoRouter(
    initialLocation: '/login',
    refreshListenable: notifier,
    redirect: notifier.redirect,
    routes: [
      // ── Auth routes (no NavigationBar) ─────────────────────────────────
      GoRoute(
        path: '/login',
        builder: (_, _) => const LoginScreen(),
      ),
      GoRoute(
        path: '/register',
        builder: (_, _) => const RegisterScreen(),
      ),
      GoRoute(
        path: '/verify-notice',
        builder: (_, state) => VerifyNoticeScreen(
          email: state.uri.queryParameters['email'],
        ),
      ),
      GoRoute(
        path: '/forgot-password',
        builder: (_, _) => const ForgotPasswordScreen(),
      ),
      // ── App routes (inside shell with NavigationBar) ────────────────────
      ShellRoute(
        builder: (context, state, child) => AppShell(child: child),
        routes: [
          GoRoute(
            path: '/vote',
            builder: (_, _) => const EventListScreen(),
            routes: [
              GoRoute(
                path: 'create',
                builder: (_, _) => const CreateEventScreen(),
              ),
            ],
          ),
          GoRoute(
            path: '/events/:id',
            builder: (_, state) => EventDetailScreen(
              eventId: state.pathParameters['id']!,
            ),
          ),
          GoRoute(
            path: '/devices',
            builder: (_, _) => const MyDevicesScreen(),
            routes: [
              GoRoute(
                path: ':id',
                builder: (_, state) => DeviceDetailScreen(
                  deviceId: state.pathParameters['id']!,
                  device: state.extra is Device ? state.extra as Device : null,
                ),
              ),
            ],
          ),
          GoRoute(
            path: '/delegation',
            builder: (_, _) => const DelegationScreen(),
          ),
          GoRoute(
            path: '/playlists',
            builder: (_, _) => const PlaylistListScreen(),
            routes: [
              GoRoute(
                path: 'create',
                builder: (_, _) => const CreatePlaylistScreen(),
              ),
            ],
          ),
          GoRoute(
            path: '/playlists/:id',
            builder: (_, _) => const PlaylistEditorScreen(),
          ),
          GoRoute(
            path: '/profile',
            builder: (_, _) => const ProfileScreen(),
            routes: [
              GoRoute(
                path: 'friends',
                builder: (_, _) => const FriendsScreen(),
              ),
              GoRoute(
                path: 'users/:id',
                builder: (_, state) => UserProfileScreen(
                  userId: state.pathParameters['id']!,
                ),
              ),
            ],
          ),
        ],
      ),
    ],
  );
});

// ── Router notifier (bridges Riverpod → GoRouter refreshListenable) ──────────

class _RouterNotifier extends ChangeNotifier {
  final Ref _ref;

  _RouterNotifier(this._ref) {
    _ref.listen<AsyncValue<AuthStatus>>(
      authProvider,
      (_, _) => notifyListeners(),
    );
  }

  String? redirect(BuildContext context, GoRouterState state) {
    final authAsync = _ref.read(authProvider);

    // Still checking stored tokens — do not redirect yet.
    if (authAsync.isLoading) return null;

    final status =
        authAsync.value ?? AuthStatus.unauthenticated;
    final path = state.uri.path;

    final isAuthRoute = path.startsWith('/login') ||
        path.startsWith('/register') ||
        path.startsWith('/verify-notice') ||
        path.startsWith('/forgot-password');

    if (status == AuthStatus.unauthenticated && !isAuthRoute) return '/login';
    if (status == AuthStatus.authenticated && isAuthRoute) return '/vote';
    return null;
  }
}

// ── App shell (bottom NavigationBar) ─────────────────────────────────────────

class AppShell extends StatelessWidget {
  const AppShell({super.key, required this.child});

  final Widget child;

  static const _tabs = [
    (icon: Icons.how_to_vote_outlined, label: 'Track Vote', path: '/vote'),
    (icon: Icons.devices_outlined, label: 'Devices', path: '/devices'),
    (icon: Icons.swap_horiz_outlined, label: 'Delegation', path: '/delegation'),
    (icon: Icons.queue_music_outlined, label: 'Playlist', path: '/playlists'),
    (icon: Icons.person_outline, label: 'Profile', path: '/profile'),
  ];

  int _currentIndex(BuildContext context) {
    final location = GoRouterState.of(context).uri.path;
    final idx = _tabs.indexWhere((t) => location.startsWith(t.path));
    return idx < 0 ? 0 : idx;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex(context),
        onDestinationSelected: (i) => context.go(_tabs[i].path),
        destinations: _tabs
            .map((t) => NavigationDestination(icon: Icon(t.icon), label: t.label))
            .toList(),
      ),
    );
  }
}
