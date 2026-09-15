import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import '../../router/app_router.dart';
import '../../theme/app_theme.dart';
import 'dashboard_nav_bar.dart';

class AwaitingAssignmentScreen extends StatelessWidget {
  const AwaitingAssignmentScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.cream,
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const SizedBox(height: 24),
              Text(
                'Hello, Admin',
                style: GoogleFonts.plusJakartaSans(
                  fontSize: 22,
                  fontWeight: FontWeight.w700,
                  color: AppColors.ink900,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                'Your group is being set up',
                style: GoogleFonts.inter(
                  fontSize: 14,
                  color: AppColors.ink400,
                ),
              ),
              const SizedBox(height: 32),
              Center(
                child: Column(
                  children: [
                    Container(
                      width: 80,
                      height: 80,
                      decoration: const BoxDecoration(
                        shape: BoxShape.circle,
                        color: AppColors.green100,
                      ),
                      child: const Icon(
                        Icons.group_rounded,
                        size: 36,
                        color: AppColors.green600,
                      ),
                    ),
                    const SizedBox(height: 16),
                    Text(
                      'Waiting for your group',
                      style: GoogleFonts.plusJakartaSans(
                        fontSize: 17,
                        fontWeight: FontWeight.w700,
                        color: AppColors.ink900,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      'A Super Admin will create your group and assign it to you. Once assigned, you can configure financial rules and start managing members.',
                      textAlign: TextAlign.center,
                      style: GoogleFonts.inter(
                        fontSize: 13,
                        color: AppColors.ink600,
                        height: 1.5,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 32),
              Text(
                'How it works',
                style: GoogleFonts.plusJakartaSans(
                  fontSize: 15,
                  fontWeight: FontWeight.w700,
                  color: AppColors.ink900,
                ),
              ),
              const SizedBox(height: 16),
              _TimelineStep(
                number: 1,
                title: 'Super Admin creates your group',
                subtitle: 'Group details are set up in the system',
                isLast: false,
              ),
              _TimelineStep(
                number: 2,
                title: 'Group is assigned to you',
                subtitle: 'You receive a notification',
                isLast: false,
              ),
              _TimelineStep(
                number: 3,
                title: 'You set up financial rules',
                subtitle: 'Configure savings, loans, and fines',
                isLast: false,
              ),
              _TimelineStep(
                number: 4,
                title: 'You add members',
                subtitle: 'Invite members to your group',
                isLast: true,
              ),
              const SizedBox(height: 24),
              Container(
                width: double.infinity,
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                decoration: BoxDecoration(
                  color: AppColors.gold100,
                  borderRadius: AppRadius.sm,
                ),
                child: Row(
                  children: [
                    const Icon(Icons.science_rounded, size: 18, color: AppColors.gold500),
                    const SizedBox(width: 8),
                    Text(
                      'Demo only',
                      style: GoogleFonts.inter(
                        fontSize: 13,
                        fontWeight: FontWeight.w600,
                        color: AppColors.gold500,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                height: 48,
                child: OutlinedButton(
                  onPressed: () {
                    Navigator.pushNamed(context, AppRouter.groupAssigned);
                  },
                  style: OutlinedButton.styleFrom(
                    foregroundColor: AppColors.teal900,
                    side: const BorderSide(color: AppColors.teal900, width: 1.5),
                    shape: RoundedRectangleBorder(
                      borderRadius: AppRadius.md,
                    ),
                  ),
                  child: Text(
                    'Simulate: group assigned',
                    style: GoogleFonts.inter(
                      fontSize: 15,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ),
              const SizedBox(height: 24),
            ],
          ),
        ),
      ),
      bottomNavigationBar: const DashboardNavBar(currentIndex: 0),
    );
  }
}

class _TimelineStep extends StatelessWidget {
  final int number;
  final String title;
  final String subtitle;
  final bool isLast;

  const _TimelineStep({
    required this.number,
    required this.title,
    required this.subtitle,
    required this.isLast,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Column(
          children: [
            Container(
              width: 20,
              height: 20,
              decoration: const BoxDecoration(
                shape: BoxShape.circle,
                color: AppColors.green600,
              ),
              child: Center(
                child: Text(
                  '$number',
                  style: GoogleFonts.inter(
                    fontSize: 10,
                    fontWeight: FontWeight.w700,
                    color: AppColors.white,
                  ),
                ),
              ),
            ),
            if (!isLast)
              Container(
                width: 2,
                height: 32,
                color: AppColors.green600,
              ),
          ],
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Padding(
            padding: const EdgeInsets.only(top: 1),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: GoogleFonts.inter(
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                    color: AppColors.ink900,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  subtitle,
                  style: GoogleFonts.inter(
                    fontSize: 12,
                    color: AppColors.ink400,
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}
