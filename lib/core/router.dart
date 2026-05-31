import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:music_room/features/track_vote/track_vote_screen.dart';
import 'package:music_room/features/delegation/delegation_screen.dart';
import 'package:music_room/features/playlist_editor/playlist_editor_screen.dart';
import 'package:music_room/features/profile/profile_screen.dart';

final router = GoRouter(
  initialLocation: '/vote',
  routes: [
    ShellRoute(
      builder: (context, state, child) => AppShell(child: child),
      routes: [
        GoRoute(path: '/vote', builder: (context, _) => const TrackVoteScreen()),
        GoRoute(path: '/delegation', builder: (context, _) => const DelegationScreen()),
        GoRoute(path: '/playlist', builder: (context, _) => const PlaylistEditorScreen()),
        GoRoute(path: '/profile', builder: (context, _) => const ProfileScreen()),
      ],
    ),
  ],
);

class AppShell extends StatelessWidget {
  const AppShell({super.key, required this.child});

  final Widget child;

  static const _tabs = [
    (icon: Icons.how_to_vote_outlined, label: 'Track Vote', path: '/vote'),
    (icon: Icons.swap_horiz_outlined, label: 'Delegation', path: '/delegation'),
    (icon: Icons.queue_music_outlined, label: 'Playlist', path: '/playlist'),
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
