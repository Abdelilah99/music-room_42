import 'package:flutter/material.dart';

class PlaylistEditorScreen extends StatelessWidget {
  const PlaylistEditorScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Playlist Editor')),
      body: const Center(child: Text('Playlist Editor')),
    );
  }
}
