import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import '../../theme/app_theme.dart';
import '../../router/app_router.dart';
import 'auth_widgets.dart';

class WelcomeScreen extends StatelessWidget {
  const WelcomeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [AppColors.teal900, AppColors.teal800],
          ),
        ),
        child: SafeArea(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 30),
            child: Column(
              children: [
                const Spacer(),
                const BrandMark(size: 70, radius: 20, fontSize: 30),
                const SizedBox(height: 18),
                Text(
                  'PesaBox',
                  style: GoogleFonts.plusJakartaSans(
                    fontSize: 21,
                    fontWeight: FontWeight.w800,
                    color: AppColors.white,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  'SIMPLE. SECURE. TOGETHER.',
                  style: GoogleFonts.inter(
                    fontSize: 12.5,
                    letterSpacing: 0.4,
                    color: const Color(0xFF8FC6AF),
                  ),
                ),
                const SizedBox(height: 18),
                Text(
                  'Manage your savings group, meetings and finances — all in one place.',
                  textAlign: TextAlign.center,
                  style: GoogleFonts.inter(
                    fontSize: 13.5,
                    height: 1.6,
                    color: const Color(0xFFBFE0D3),
                  ),
                ),
                const Spacer(),
                PrimaryButton(
                  text: 'Get started',
                  onPressed: () =>
                      Navigator.pushNamed(context, AppRouter.register),
                ),
                const SizedBox(height: 10),
                OutlineButton(
                  text: 'Login',
                  onPressed: () => Navigator.pushNamed(context, AppRouter.login),
                ),
                const SizedBox(height: 40),
              ],
            ),
          ),
        ),
      ),
    );
  }
}