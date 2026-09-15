import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:google_fonts/google_fonts.dart';

import '../../theme/app_theme.dart';
import '../../router/app_router.dart';
import 'auth_widgets.dart';

class OtpScreen extends StatefulWidget {
  const OtpScreen({super.key});

  @override
  State<OtpScreen> createState() => _OtpScreenState();
}

class _OtpScreenState extends State<OtpScreen> {
  final List<TextEditingController> _controllers =
      List.generate(6, (_) => TextEditingController());
  final List<FocusNode> _focusNodes = List.generate(6, (_) => FocusNode());

  Timer? _timer;
  int _secondsRemaining = 45;

  @override
  void initState() {
    super.initState();
    _controllers[0].text = '1';
    _controllers[1].text = '2';
    _startTimer();
  }

  void _startTimer() {
    _timer?.cancel();
    setState(() => _secondsRemaining = 45);
    _timer = Timer.periodic(const Duration(seconds: 1), (timer) {
      if (_secondsRemaining <= 1) {
        timer.cancel();
        setState(() => _secondsRemaining = 0);
      } else {
        setState(() => _secondsRemaining--);
      }
    });
  }

  String _format(int totalSeconds) {
    final minutes = (totalSeconds ~/ 60).toString().padLeft(2, '0');
    final seconds = (totalSeconds % 60).toString().padLeft(2, '0');
    return '$minutes:$seconds';
  }

  @override
  void dispose() {
    _timer?.cancel();
    for (final controller in _controllers) {
      controller.dispose();
    }
    for (final node in _focusNodes) {
      node.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.cream,
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(18, 6, 18, 32),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const AuthHeader(),
              const SizedBox(height: 24),
              Text(
                'Verify your phone number',
                style: GoogleFonts.plusJakartaSans(
                  fontSize: 19,
                  fontWeight: FontWeight.w800,
                  color: AppColors.ink900,
                ),
              ),
              const SizedBox(height: 6),
              Text.rich(
                TextSpan(
                  text: "We've sent a 6-digit code to\n",
                  style: GoogleFonts.inter(
                    fontSize: 13,
                    height: 1.5,
                    color: AppColors.ink600,
                  ),
                  children: [
                    TextSpan(
                      text: '+255 7xx xxx xxx',
                      style: GoogleFonts.inter(
                        fontSize: 13,
                        height: 1.5,
                        fontWeight: FontWeight.w700,
                        color: AppColors.ink900,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 22),
              Row(
                children: [
                  for (var i = 0; i < 6; i++)
                    Expanded(
                      child: Padding(
                        padding: EdgeInsets.only(left: i == 0 ? 0 : 8),
                        child: Container(
                          height: 52,
                          decoration: BoxDecoration(
                            color: AppColors.white,
                            borderRadius: BorderRadius.circular(12),
                            border: Border.all(
                              color: AppColors.line,
                              width: 1.5,
                            ),
                          ),
                          child: TextField(
                            controller: _controllers[i],
                            focusNode: _focusNodes[i],
                            textAlign: TextAlign.center,
                            keyboardType: TextInputType.number,
                            maxLength: 1,
                            inputFormatters: [
                              FilteringTextInputFormatter.digitsOnly,
                            ],
                            style: GoogleFonts.inter(
                              fontSize: 18,
                              fontWeight: FontWeight.w700,
                              color: AppColors.ink900,
                            ),
                            decoration: const InputDecoration(
                              counterText: '',
                              border: InputBorder.none,
                              contentPadding: EdgeInsets.zero,
                            ),
                            onChanged: (value) {
                              if (value.isNotEmpty &&
                                  i < _controllers.length - 1) {
                                FocusScope.of(context)
                                    .requestFocus(_focusNodes[i + 1]);
                              } else if (value.isEmpty && i > 0) {
                                FocusScope.of(context)
                                    .requestFocus(_focusNodes[i - 1]);
                              }
                            },
                          ),
                        ),
                      ),
                    ),
                ],
              ),
              const SizedBox(height: 14),
              Center(
                child: _secondsRemaining > 0
                    ? Text(
                        'Resend code (${_format(_secondsRemaining)})',
                        style: GoogleFonts.inter(
                          fontSize: 12.5,
                          color: AppColors.ink400,
                        ),
                      )
                    : GestureDetector(
                        onTap: _startTimer,
                        child: Padding(
                          padding: const EdgeInsets.all(4),
                          child: Text(
                            'Resend code',
                            style: GoogleFonts.inter(
                              fontSize: 12.5,
                              fontWeight: FontWeight.w700,
                              color: AppColors.green600,
                            ),
                          ),
                        ),
                      ),
              ),
              const SizedBox(height: 24),
              PrimaryButton(
                text: 'Verify',
                onPressed: () =>
                    Navigator.pushNamed(context, AppRouter.useCase),
              ),
            ],
          ),
        ),
      ),
    );
  }
}